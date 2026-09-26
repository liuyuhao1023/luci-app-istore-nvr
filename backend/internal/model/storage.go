package model

import "time"

// StorageType 存储类型
type StorageType string

const (
	StorageTypeLocal StorageType = "local" // 本地磁盘存储 (例如挂载的数据盘)
	StorageTypeSMB   StorageType = "smb"   // SMB/CIFS 网络共享 (NAS / Windows共享)
)

// StorageStatus 存储挂载状态
type StorageStatus string

const (
	StorageStatusMounted   StorageStatus = "mounted"   // 正常挂载并具备读写权限
	StorageStatusOffline   StorageStatus = "offline"   // 离线/无法连接
	StorageStatusError     StorageStatus = "error"     // 挂载或权限异常
	StorageStatusUnmounted StorageStatus = "unmounted" // 未挂载
)

// Storage 存储设备配置与状态模型
type Storage struct {
	ID             uint          `json:"id" gorm:"primaryKey"`
	Name           string        `json:"name" gorm:"size:128;not null"`          // 存储名称 (例如: NAS01录像存储, 本地SSD)
	Type           StorageType   `json:"type" gorm:"size:32;default:'local'"`    // local / smb
	
	// 本地路径或挂载目标路径
	MountPoint     string        `json:"mount_point" gorm:"size:256;not null"`   // 挂载点或本地录像根目录，如 /mnt/sda1/recordings 或 /mnt/nvr/storage/nas01
	
	// SMB专属配置
	ServerHost     string        `json:"server_host" gorm:"size:128"`            // SMB服务器IP或主机名 (例如 192.168.1.20)
	ShareName      string        `json:"share_name" gorm:"size:128"`             // 共享目录名 (例如 CameraRecordings)
	Username       string        `json:"username" gorm:"size:64"`                // SMB访问账号
	PasswordEnc    string        `json:"-" gorm:"size:256"`                      // 加密密码 (严禁明文暴露)
	SMBVersion     string        `json:"smb_version" gorm:"size:32;default:'3.0'"` // SMB协议版本: auto, 3.0, 2.1
	MountOptions   string        `json:"mount_options" gorm:"size:256"`          // 额外挂载参数
	
	// 容量与运行指标
	Status         StorageStatus `json:"status" gorm:"default:'unmounted'"`      // 当前状态
	TotalBytes     uint64        `json:"total_bytes"`                            // 总容量 (字节)
	UsedBytes      uint64        `json:"used_bytes"`                             // 已使用容量 (字节)
	FreeBytes      uint64        `json:"free_bytes"`                             // 剩余可用容量 (字节)
	WriteSpeedKBps float64       `json:"write_speed_kbps"`                       // 当前实时写入速度 KB/s
	AllocatedCameras int         `json:"allocated_cameras"`                      // 已分配的摄像头总数
	
	// 安全与保留策略
	AlertThresholdPercent int    `json:"alert_threshold_percent" gorm:"default:90"` // 空间告警阈值百分比 (如 90%)
	AutoCleanEnabled      bool   `json:"auto_clean_enabled" gorm:"default:true"`    // 是否启用空间不足自动清理最早普通录像
	MinRetentionDays      int    `json:"min_retention_days" gorm:"default:3"`       // 最短保护天数 (防止误删近期录像)
	LastErrorMessage      string `json:"last_error_message" gorm:"size:256"`        // 最近异常记录
	LastCheckedTime       *time.Time `json:"last_checked_time"`                     // 最近健康检测时间

	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}
