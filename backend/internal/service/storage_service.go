package service

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"gorm.io/gorm"

	"istore-nvr/internal/model"
	"istore-nvr/internal/utils"
)

type StorageService struct {
	db *gorm.DB
}

func NewStorageService(db *gorm.DB) *StorageService {
	return &StorageService{db: db}
}

// ListStorages 获取所有存储设备列表 (并刷新容量和状态)
func (s *StorageService) ListStorages() ([]model.Storage, error) {
	var storages []model.Storage
	if err := s.db.Find(&storages).Error; err != nil {
		return nil, err
	}

	for i := range storages {
		s.refreshStorageStats(&storages[i])
		_ = s.db.Save(&storages[i]).Error
	}

	return storages, nil
}

// GetStorageByID 获取单个存储详情
func (s *StorageService) GetStorageByID(id uint) (*model.Storage, string, error) {
	var st model.Storage
	if err := s.db.First(&st, id).Error; err != nil {
		return nil, "", err
	}
	plainPass, _ := utils.DecryptPassword(st.PasswordEnc)
	s.refreshStorageStats(&st)
	return &st, plainPass, nil
}

// AddStorage 添加本地或 SMB 网络存储
func (s *StorageService) AddStorage(req *model.Storage, rawPassword string) (*model.Storage, error) {
	if req.Name == "" {
		return nil, errors.New("存储名称不能为空")
	}

	if req.Type == model.StorageTypeSMB {
		if req.ServerHost == "" || req.ShareName == "" {
			return nil, errors.New("SMB 服务器地址和共享目录名不能为空")
		}
		if req.MountPoint == "" {
			// 自动分配标准挂载路径
			cleanName := strings.ReplaceAll(strings.ToLower(req.Name), " ", "_")
			req.MountPoint = fmt.Sprintf("/mnt/nvr/storage/%s", cleanName)
		}
		if req.SMBVersion == "" {
			req.SMBVersion = "3.0"
		}
		if rawPassword != "" {
			enc, err := utils.EncryptPassword(rawPassword)
			if err != nil {
				return nil, err
			}
			req.PasswordEnc = enc
		}
	} else {
		// 本地存储
		if req.MountPoint == "" {
			req.MountPoint = "/mnt/sata1-4/recordings"
		}
		// 严禁将录像根目录设为根目录 / 或 /overlay
		if req.MountPoint == "/" || strings.HasPrefix(req.MountPoint, "/overlay") {
			return nil, errors.New("安全保护：严禁将软路由根目录或 /overlay 分区设置为录像存储目录！")
		}
	}

	if err := s.db.Create(req).Error; err != nil {
		return nil, err
	}

	// 若为 SMB 存储，尝试执行挂载与读写测试
	if req.Type == model.StorageTypeSMB {
		go func(st model.Storage, pass string) {
			_ = s.MountSMB(&st, pass)
			_ = s.db.Save(&st).Error
		}(*req, rawPassword)
	} else {
		s.refreshStorageStats(req)
		_ = s.db.Save(req).Error
	}

	return req, nil
}

// UpdateStorage 更新存储配置
func (s *StorageService) UpdateStorage(id uint, req *model.Storage, rawPassword string) (*model.Storage, error) {
	var st model.Storage
	if err := s.db.First(&st, id).Error; err != nil {
		return nil, err
	}

	st.Name = req.Name
	st.AlertThresholdPercent = req.AlertThresholdPercent
	st.AutoCleanEnabled = req.AutoCleanEnabled
	st.MinRetentionDays = req.MinRetentionDays

	if st.Type == model.StorageTypeSMB {
		st.ServerHost = req.ServerHost
		st.ShareName = req.ShareName
		st.Username = req.Username
		st.SMBVersion = req.SMBVersion
		if rawPassword != "" {
			enc, err := utils.EncryptPassword(rawPassword)
			if err != nil {
				return nil, err
			}
			st.PasswordEnc = enc
		}
	}

	if err := s.db.Save(&st).Error; err != nil {
		return nil, err
	}

	return &st, nil
}

// DeleteStorage 删除存储配置 (有分配摄像头或录像文件时提示)
func (s *StorageService) DeleteStorage(id uint) error {
	var st model.Storage
	if err := s.db.First(&st, id).Error; err != nil {
		return err
	}

	// 检查是否有摄像头正在使用
	var camCount int64
	s.db.Model(&model.Camera{}).Where("storage_id = ?", id).Count(&camCount)
	if camCount > 0 {
		return fmt.Errorf("当前仍有 %d 台摄像头关联该存储，请先将其切换到其他存储设备", camCount)
	}

	// 若是 SMB 挂载，卸载该挂载点
	if st.Type == model.StorageTypeSMB && st.Status == model.StorageStatusMounted {
		_ = s.UnmountSMB(&st)
	}

	return s.db.Delete(&st).Error
}

// MountSMB 执行安全 SMB 挂载 (使用专用凭证文件，杜绝命令行密码泄漏)
func (s *StorageService) MountSMB(st *model.Storage, rawPassword string) error {
	if runtime.GOOS != "linux" {
		// 在 Windows 开发环境下模拟挂载就绪状态
		st.Status = model.StorageStatusMounted
		return nil
	}

	if rawPassword == "" && st.PasswordEnc != "" {
		rawPassword, _ = utils.DecryptPassword(st.PasswordEnc)
	}

	// 1. 确保挂载点目录存在
	if err := os.MkdirAll(st.MountPoint, 0755); err != nil {
		st.Status = model.StorageStatusError
		st.LastErrorMessage = fmt.Sprintf("创建挂载目录失败: %v", err)
		return err
	}

	// 2. 创建受限权限凭证临时文件 (0600)
	credFile, err := os.CreateTemp("/tmp", "cifs_cred_*")
	if err != nil {
		st.Status = model.StorageStatusError
		st.LastErrorMessage = fmt.Sprintf("创建临时凭据文件失败: %v", err)
		return err
	}
	credPath := credFile.Name()
	defer os.Remove(credPath)

	credContent := fmt.Sprintf("username=%s\npassword=%s\n", st.Username, rawPassword)
	if _, err := credFile.WriteString(credContent); err != nil {
		_ = credFile.Close()
		return err
	}
	_ = credFile.Close()

	// 3. 构建 mount.cifs 命令
	smbVersion := st.SMBVersion
	if smbVersion == "" || smbVersion == "auto" {
		smbVersion = "3.0"
	}

	uncPath := fmt.Sprintf("//%s/%s", st.ServerHost, st.ShareName)
	options := fmt.Sprintf("credentials=%s,vers=%s,iocharset=utf8,file_mode=0777,dir_mode=0777,soft", credPath, smbVersion)

	// 先尝试 umount 避免残留挂载
	_ = exec.Command("umount", "-l", st.MountPoint).Run()

	cmd := exec.Command("mount", "-t", "cifs", uncPath, st.MountPoint, "-o", options)
	output, err := cmd.CombinedOutput()
	now := time.Now()
	st.LastCheckedTime = &now

	if err != nil {
		st.Status = model.StorageStatusError
		st.LastErrorMessage = fmt.Sprintf("SMB 挂载失败: %s (%s)", string(output), err.Error())
		log.Printf("[SMB] 挂载 %s 失败: %s", uncPath, string(output))
		return fmt.Errorf("挂载失败: %s", string(output))
	}

	// 4. 读写权限验证 (第17.2节要求: 读写测试)
	testFile := filepath.Join(st.MountPoint, ".istore_nvr_rw_test")
	if writeErr := os.WriteFile(testFile, []byte("ok"), 0644); writeErr != nil {
		_ = exec.Command("umount", st.MountPoint).Run()
		st.Status = model.StorageStatusError
		st.LastErrorMessage = fmt.Sprintf("SMB 共享目录无写入权限: %v", writeErr)
		return writeErr
	}
	_ = os.Remove(testFile)

	st.Status = model.StorageStatusMounted
	st.LastErrorMessage = ""
	s.refreshStorageStats(st)
	log.Printf("[SMB] 存储设备 [%s] 成功挂载至 %s", st.Name, st.MountPoint)
	return nil
}

// UnmountSMB 卸载 SMB 网络存储
func (s *StorageService) UnmountSMB(st *model.Storage) error {
	if runtime.GOOS != "linux" {
		st.Status = model.StorageStatusUnmounted
		return nil
	}
	cmd := exec.Command("umount", "-l", st.MountPoint)
	_ = cmd.Run()
	st.Status = model.StorageStatusUnmounted
	return nil
}

// ValidateStorageSafe 核心安全保护函数 (第17.6节要求): 确认存储处于挂载就绪状态，严禁写入未挂载的空本地目录
func (s *StorageService) ValidateStorageSafe(st *model.Storage) error {
	if st == nil {
		return errors.New("存储配置对象为空")
	}

	if st.Type == model.StorageTypeLocal {
		if st.MountPoint == "/" || strings.HasPrefix(st.MountPoint, "/overlay") {
			return errors.New("系统保护红线：禁止使用根目录或 /overlay 分区")
		}
		if _, err := os.Stat(st.MountPoint); os.IsNotExist(err) {
			return fmt.Errorf("本地存储目录 %s 不存在", st.MountPoint)
		}
		return nil
	}

	// SMB 类型检查
	if st.Status != model.StorageStatusMounted {
		return fmt.Errorf("SMB 存储 [%s] 当前处于离线/未挂载状态，为保护软路由根目录，已暂停录像写入", st.Name)
	}

	// 进一步检查 Linux 挂载状态
	if runtime.GOOS == "linux" {
		isMounted, err := checkIfMountPoint(st.MountPoint)
		if err != nil || !isMounted {
			st.Status = model.StorageStatusOffline
			st.LastErrorMessage = "检测到 SMB 挂载已脱机，触发熔断保护"
			s.db.Save(st)
			return fmt.Errorf("熔断保护：目标目录 %s 未实际挂载 SMB，拒绝写入本地空目录", st.MountPoint)
		}
	}

	return nil
}

// checkIfMountPoint 检查指定路径是否为独立挂载点
func checkIfMountPoint(path string) (bool, error) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == path {
			return true, nil
		}
	}
	return false, nil
}

// refreshStorageStats 刷新存储容量信息
func (s *StorageService) refreshStorageStats(st *model.Storage) {
	// 统计关联摄像头数
	var count int64
	s.db.Model(&model.Camera{}).Where("storage_id = ?", st.ID).Count(&count)
	st.AllocatedCameras = int(count)

	if _, err := os.Stat(st.MountPoint); os.IsNotExist(err) {
		return
	}

	// 在 Linux 上获取 statfs
	if runtime.GOOS == "linux" {
		var stat syscall.Statfs_t
		if err := syscall.Statfs(st.MountPoint, &stat); err == nil {
			st.TotalBytes = stat.Blocks * uint64(stat.Bsize)
			st.FreeBytes = stat.Bavail * uint64(stat.Bsize)
			st.UsedBytes = st.TotalBytes - st.FreeBytes
		}
	} else {
		// 开发机模拟数据
		st.TotalBytes = 100 * 1024 * 1024 * 1024
		st.UsedBytes = 20 * 1024 * 1024 * 1024
		st.FreeBytes = 80 * 1024 * 1024 * 1024
	}
}
