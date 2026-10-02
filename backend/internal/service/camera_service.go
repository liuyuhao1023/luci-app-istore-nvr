package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	"istore-nvr/internal/adapter"
	"istore-nvr/internal/model"
	"istore-nvr/internal/utils"
)

type CameraService struct {
	db              *gorm.DB
	discoveryEngine *adapter.DiscoveryEngine
	
	// 运行期状态回调函数 (解耦录像引擎与流媒体引擎)
	onCameraOnline  func(cam *model.Camera)
	onCameraOffline func(cam *model.Camera)
}

func NewCameraService(db *gorm.DB) *CameraService {
	return &CameraService{
		db:              db,
		discoveryEngine: adapter.NewDiscoveryEngine(),
	}
}

func (s *CameraService) SetCallbacks(onOnline, onOffline func(cam *model.Camera)) {
	s.onCameraOnline = onOnline
	s.onCameraOffline = onOffline
}

// ListCameras 获取摄像头列表 (支持按分组、在线状态、关键字筛选)
func (s *CameraService) ListCameras(group string, onlineOnly *bool, keyword string) ([]model.CameraDTO, error) {
	var cameras []model.Camera
	query := s.db.Model(&model.Camera{})

	if group != "" {
		query = query.Where("`group` = ?", group)
	}
	if onlineOnly != nil {
		query = query.Where("is_online = ?", *onlineOnly)
	}
	if keyword != "" {
		kw := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR ip LIKE ? OR model LIKE ?", kw, kw, kw)
	}

	if err := query.Order("id asc").Find(&cameras).Error; err != nil {
		return nil, err
	}

	dtos := make([]model.CameraDTO, len(cameras))
	for i, cam := range cameras {
		dtos[i] = model.CameraDTO{
			Camera:      cam,
			HasPassword: cam.PasswordEnc != "",
		}
	}
	return dtos, nil
}

// GetCameraByID 获取单个摄像头详情
func (s *CameraService) GetCameraByID(id uint) (*model.Camera, string, error) {
	var cam model.Camera
	if err := s.db.First(&cam, id).Error; err != nil {
		return nil, "", err
	}

	plainPass, _ := utils.DecryptPassword(cam.PasswordEnc)
	return &cam, plainPass, nil
}

// AddCamera 手动添加摄像头
func (s *CameraService) AddCamera(req *model.Camera, rawPassword string) (*model.Camera, error) {
	if req.IP == "" {
		return nil, errors.New("摄像头 IP 地址不能为空")
	}
	if req.Name == "" {
		req.Name = fmt.Sprintf("摄像头-%s", req.IP)
	}

	// 密码加密存储
	if rawPassword != "" {
		enc, err := utils.EncryptPassword(rawPassword)
		if err != nil {
			return nil, fmt.Errorf("密码加密失败: %w", err)
		}
		req.PasswordEnc = enc
	}

	if req.Brand == "" || req.Brand == "auto" {
		req.Brand = adapter.DetectBrand(req.IP, req.HTTPPort, req.RTSPPort)
	}
	adp := adapter.GetAdapter(req.Brand)

	// 自动生成对应品牌规范的主/子码流 URL (若未显式指定)
	if req.MainStreamURL == "" || req.SubStreamURL == "" {
		mainURL, subURL := adp.BuildStreamURLs(req, rawPassword)
		if req.MainStreamURL == "" {
			req.MainStreamURL = mainURL
		}
		if req.SubStreamURL == "" {
			req.SubStreamURL = subURL
		}
	}

	if err := s.db.Create(req).Error; err != nil {
		return nil, err
	}

	// 异步发起首次快速探测
	go s.TestAndRefreshStatus(req.ID)

	return req, nil
}

// UpdateCamera 更新摄像头配置
func (s *CameraService) UpdateCamera(id uint, updateData *model.Camera, rawPassword string) (*model.Camera, error) {
	var cam model.Camera
	if err := s.db.First(&cam, id).Error; err != nil {
		return nil, err
	}

	cam.Name = updateData.Name
	cam.IP = updateData.IP
	if updateData.Brand != "" {
		cam.Brand = updateData.Brand
	}
	cam.Subnet = updateData.Subnet
	cam.NetworkInterface = updateData.NetworkInterface
	cam.RTSPPort = updateData.RTSPPort
	cam.HTTPPort = updateData.HTTPPort
	cam.Username = updateData.Username
	cam.Group = updateData.Group
	cam.Channel = updateData.Channel
	cam.TransportMode = updateData.TransportMode
	cam.RecordEnabled = updateData.RecordEnabled
	cam.RecordMode = updateData.RecordMode
	cam.StorageID = updateData.StorageID
	cam.RetentionDays = updateData.RetentionDays
	cam.SegmentMinutes = updateData.SegmentMinutes

	if rawPassword != "" {
		enc, err := utils.EncryptPassword(rawPassword)
		if err != nil {
			return nil, err
		}
		cam.PasswordEnc = enc
	}

	// 优先尊重用户在前端输入的自定义或修改后的码流
	if updateData.MainStreamURL != "" {
		cam.MainStreamURL = updateData.MainStreamURL
	}
	if updateData.SubStreamURL != "" {
		cam.SubStreamURL = updateData.SubStreamURL
	}

	// 若未指定码流地址，根据当前品牌适配器重新生成
	if cam.MainStreamURL == "" || cam.SubStreamURL == "" {
		plainPass, _ := utils.DecryptPassword(cam.PasswordEnc)
		adp := adapter.GetAdapter(cam.Brand)
		mainURL, subURL := adp.BuildStreamURLs(&cam, plainPass)
		if cam.MainStreamURL == "" {
			cam.MainStreamURL = mainURL
		}
		if cam.SubStreamURL == "" {
			cam.SubStreamURL = subURL
		}
	}

	if err := s.db.Save(&cam).Error; err != nil {
		return nil, err
	}

	if s.onCameraOnline != nil {
		s.onCameraOnline(&cam)
	}

	// 触发一次连接测试并刷新流媒体状态
	go s.TestAndRefreshStatus(cam.ID)

	return &cam, nil
}

// DeleteCamera 删除摄像头
func (s *CameraService) DeleteCamera(id uint) error {
	var cam model.Camera
	if err := s.db.First(&cam, id).Error; err != nil {
		return err
	}

	// 通知停止录像与拉流
	if s.onCameraOffline != nil {
		s.onCameraOffline(&cam)
	}

	return s.db.Delete(&cam).Error
}

// BatchUpdateRecord 批量修改录像开关
func (s *CameraService) BatchUpdateRecord(ids []uint, enabled bool) error {
	return s.db.Model(&model.Camera{}).Where("id IN ?", ids).Update("record_enabled", enabled).Error
}

// BatchUpdateStorage 批量修改摄像头的录像存储位置 (第17.4节需求)
func (s *CameraService) BatchUpdateStorage(ids []uint, storageID uint) error {
	// 校验目标存储是否存在
	var storage model.Storage
	if err := s.db.First(&storage, storageID).Error; err != nil {
		return fmt.Errorf("指定的存储设备不存在: %w", err)
	}

	return s.db.Model(&model.Camera{}).Where("id IN ?", ids).Update("storage_id", storageID).Error
}

// TestConnection 跨网段连接测试 (供 API 交互即时调用)
func (s *CameraService) TestConnection(cam *model.Camera, rawPassword string) (*adapter.DeviceInfo, error) {
	if rawPassword == "" && cam.PasswordEnc != "" {
		rawPassword, _ = utils.DecryptPassword(cam.PasswordEnc)
	}
	if cam.Brand == "" || cam.Brand == "auto" {
		cam.Brand = adapter.DetectBrand(cam.IP, cam.HTTPPort, cam.RTSPPort)
	}
	adp := adapter.GetAdapter(cam.Brand)
	return adp.ConnectTest(cam, rawPassword)
}

// TestAndRefreshStatus 测试并刷新数据库中设备状态与元数据
func (s *CameraService) TestAndRefreshStatus(id uint) (*model.Camera, error) {
	var cam model.Camera
	if err := s.db.First(&cam, id).Error; err != nil {
		return nil, err
	}

	if cam.Brand == "" || cam.Brand == "auto" {
		cam.Brand = adapter.DetectBrand(cam.IP, cam.HTTPPort, cam.RTSPPort)
	}
	adp := adapter.GetAdapter(cam.Brand)

	plainPass, _ := utils.DecryptPassword(cam.PasswordEnc)
	info, err := adp.ConnectTest(&cam, plainPass)

	now := time.Now()
	if err != nil {
		cam.IsOnline = false
		cam.LastOfflineTime = &now
		cam.LastOfflineReason = err.Error()
		s.db.Save(&cam)

		if s.onCameraOffline != nil {
			s.onCameraOffline(&cam)
		}
		return &cam, err
	}

	wasOffline := !cam.IsOnline
	cam.IsOnline = true
	cam.LastOnlineTime = &now
	cam.LastOfflineReason = ""
	if info.Brand != "" {
		cam.Brand = info.Brand
	}
	if info.Model != "" {
		cam.Model = info.Model
	}
	if info.SerialNumber != "" {
		cam.SerialNumber = info.SerialNumber
	}
	if info.FirmwareVersion != "" {
		cam.FirmwareVersion = info.FirmwareVersion
	}
	if info.VideoCodec != "" {
		cam.VideoCodec = info.VideoCodec
	}
	if info.Resolution != "" {
		cam.Resolution = info.Resolution
	}
	if info.FrameRate > 0 {
		cam.FrameRate = info.FrameRate
	}
	if info.RTSPPort > 0 && info.RTSPPort != cam.RTSPPort {
		cam.RTSPPort = info.RTSPPort
	}
	if cam.MainStreamURL == "" && info.MainStreamURL != "" {
		cam.MainStreamURL = info.MainStreamURL
	}
	if cam.SubStreamURL == "" && info.SubStreamURL != "" {
		cam.SubStreamURL = info.SubStreamURL
	}

	s.db.Save(&cam)

	// 若之前离线、现已恢复，或检测到配置更新，触发在线与流媒体回调
	if (wasOffline || info.RTSPPort > 0) && s.onCameraOnline != nil {
		s.onCameraOnline(&cam)
	}

	return &cam, nil
}

// StartHeartbeatLoop 启动后台心跳协程池，定期健康探测 32~64 路摄像头
func (s *CameraService) StartHeartbeatLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.pollAllCameras()
			}
		}
	}()
}

// pollAllCameras 并发轮询所有摄像头的存活状态
func (s *CameraService) pollAllCameras() {
	var cameras []model.Camera
	if err := s.db.Find(&cameras).Error; err != nil || len(cameras) == 0 {
		return
	}

	// 控制最大并发探测数，避免突发占满软路由网络连接
	semaphore := make(chan struct{}, 16)
	var wg sync.WaitGroup

	for _, cam := range cameras {
		wg.Add(1)
		go func(c model.Camera) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			_, _ = s.TestAndRefreshStatus(c.ID)
		}(cam)
	}

	wg.Wait()
}

// ScanDevices 跨网段扫描与自动探测
func (s *CameraService) ScanDevices(ctx context.Context, subnets []string, useMulticast bool) ([]adapter.DiscoveredDevice, error) {
	var all []adapter.DiscoveredDevice
	resultsMap := make(map[string]adapter.DiscoveredDevice)

	// 1. 如果勾选或未指定网段，执行本地组播发现
	if useMulticast || len(subnets) == 0 {
		multicastList, err := s.discoveryEngine.ProbeMulticast(3)
		if err == nil {
			for _, dev := range multicastList {
				resultsMap[dev.IP] = dev
			}
		}
	}

	// 2. 如果提供了网段列表，执行并发单播跨网段扫描
	if len(subnets) > 0 {
		subnetList, err := s.discoveryEngine.ScanSubnets(ctx, subnets, 25)
		if err == nil {
			for _, dev := range subnetList {
				if _, exists := resultsMap[dev.IP]; !exists {
					resultsMap[dev.IP] = dev
				}
			}
		}
	}

	for _, dev := range resultsMap {
		all = append(all, dev)
	}
	return all, nil
}
