package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
			return nil, errors.New("本地存储必须指定存储目录路径")
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

	total, free, err := getDiskSpace(st.MountPoint)
	if err == nil && total > 0 {
		st.TotalBytes = total
		st.FreeBytes = free
		st.UsedBytes = total - free
	}
}

// CheckAndCleanStorage 核心循环录像空间回收机制：当存储空间不足或达到阈值时，自动清理最早的历史录像
func (s *StorageService) CheckAndCleanStorage(st *model.Storage) (int, int64, error) {
	if st == nil || !st.AutoCleanEnabled {
		return 0, 0, nil
	}

	s.refreshStorageStats(st)
	if st.TotalBytes == 0 {
		return 0, 0, nil
	}

	threshold := st.AlertThresholdPercent
	if threshold <= 0 || threshold > 99 {
		threshold = 90
	}

	usedPct := float64(st.UsedBytes) / float64(st.TotalBytes) * 100.0

	// 安全保底可用空间：默认保留至少 2GB 可用空间（若总容量较小则保留 10%）
	minFreeBytes := uint64(2 * 1024 * 1024 * 1024)
	if st.TotalBytes < 10*1024*1024*1024 {
		minFreeBytes = st.TotalBytes / 10
	}

	needClean := usedPct >= float64(threshold) || st.FreeBytes < minFreeBytes
	if !needClean {
		return 0, 0, nil
	}

	log.Printf("[AutoClean] 存储 [%s] (ID=%d) 触发循环空间回收: 已用 %.1f%% (阈值 %d%%), 剩余 %.2f GB (保底 %.2f GB)",
		st.Name, st.ID, usedPct, threshold,
		float64(st.FreeBytes)/1024/1024/1024, float64(minFreeBytes)/1024/1024/1024)

	var totalDeletedCount int
	var totalDeletedBytes int64

	// 目标水位：清理到使用率低于 (threshold - 5)% 且可用空间恢复到保底的 1.5 倍
	targetPct := float64(threshold - 5)
	if targetPct < 50 {
		targetPct = 50
	}
	targetFreeBytes := minFreeBytes + minFreeBytes/2

	// 循环分批清理
	for {
		// 先尝试清理超出保护期 (MinRetentionDays) 的最早录像
		var candidates []model.RecordingIndex
		query := s.db.Where("storage_id = ? AND is_protected = ?", st.ID, false)

		if st.MinRetentionDays > 0 {
			cutoff := time.Now().AddDate(0, 0, -st.MinRetentionDays)
			query = query.Where("start_time < ?", cutoff)
		}

		if err := query.Order("start_time asc").Limit(50).Find(&candidates).Error; err != nil {
			log.Printf("[AutoClean] 查询待清理录像失败: %v", err)
			break
		}

		// 如果保护期内无录像可清，但磁盘使用率已超过 95% 或剩余空间严重不足，为了保证录制不中断，强制清理未保护的最早录像
		if len(candidates) == 0 && (usedPct >= 95.0 || st.FreeBytes < minFreeBytes/2) {
			log.Printf("[AutoClean] 存储 [%s] 空间极度告急，放宽保护期限制以确保持续录像", st.Name)
			_ = s.db.Where("storage_id = ? AND is_protected = ?", st.ID, false).
				Order("start_time asc").Limit(50).Find(&candidates).Error
		}

		if len(candidates) == 0 {
			// 没有更多可清理的录像
			break
		}

		batchDeleted := 0
		for _, rec := range candidates {
			absPath := filepath.Join(st.MountPoint, rec.FilePath)
			// 删除物理文件
			if err := os.Remove(absPath); err == nil || os.IsNotExist(err) {
				totalDeletedBytes += rec.SizeBytes
				totalDeletedCount++
				batchDeleted++

				// 尝试清理可能遗留的空日期目录
				cleanEmptyParents(filepath.Dir(absPath), filepath.Join(st.MountPoint, "recordings"))
			} else {
				log.Printf("[AutoClean] 删除录像文件失败 %s: %v", absPath, err)
			}

			// 从数据库删除索引记录
			s.db.Delete(&rec)
		}

		// 刷新最新容量状态
		s.refreshStorageStats(st)
		usedPct = float64(st.UsedBytes) / float64(st.TotalBytes) * 100.0

		// 判断是否已达标
		if usedPct < targetPct && st.FreeBytes >= targetFreeBytes {
			break
		}

		if batchDeleted == 0 {
			// 如果本批次未能成功删除任何文件，避免死循环
			break
		}
	}

	_ = s.db.Save(st).Error

	if totalDeletedCount > 0 {
		log.Printf("[AutoClean] 存储 [%s] 循环回收完成: 共清理 %d 个最早录像切片，释放 %.2f MB 空间，当前可用 %.2f GB",
			st.Name, totalDeletedCount, float64(totalDeletedBytes)/1024/1024, float64(st.FreeBytes)/1024/1024/1024)

		// 记录系统告警事件
		_ = s.db.Create(&model.AlertEvent{
			Level:     "info",
			Source:    "storage",
			SourceID:  st.ID,
			Message:   fmt.Sprintf("存储 [%s] 达到循环录像阈值(%d%%)，已自动清理 %d 个历史切片，释放 %.2f MB", st.Name, threshold, totalDeletedCount, float64(totalDeletedBytes)/1024/1024),
			CreatedAt: time.Now(),
		}).Error
	}

	return totalDeletedCount, totalDeletedBytes, nil
}

// cleanEmptyParents 递归向上清理空录像目录，直到到达根录像目录为止
func cleanEmptyParents(dir, stopDir string) {
	for {
		if dir == "" || dir == "." || dir == "/" || dir == stopDir || !strings.HasPrefix(dir, stopDir) {
			break
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		_ = os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

// StartAutoCleanLoop 启动后台循环录像空间监测守护协程
func (s *StorageService) StartAutoCleanLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	go func() {
		log.Printf("[AutoClean] 存储循环录像自动回收服务已启动 (检测周期 %v)", interval)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[AutoClean] 存储循环录像自动回收服务已停止")
				return
			case <-ticker.C:
				var storages []model.Storage
				if err := s.db.Find(&storages).Error; err == nil {
					for i := range storages {
						if storages[i].AutoCleanEnabled {
							_, _, _ = s.CheckAndCleanStorage(&storages[i])
						}
					}
				}
			}
		}
	}()
}
