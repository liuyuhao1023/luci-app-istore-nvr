package model

import (
	"time"
)

// RecordMode 录像模式
type RecordMode string

const (
	RecordModeDisabled   RecordMode = "disabled"   // 关闭录像
	RecordModeContinuous RecordMode = "continuous" // 持续录像
	RecordModeScheduled  RecordMode = "scheduled"  // 定时录像
)

// StreamTransport RTSP传输协议
type StreamTransport string

const (
	TransportTCP StreamTransport = "tcp" // TCP Interleaved (跨网段首选，稳定防丢包)
	TransportUDP StreamTransport = "udp" // UDP
)

// Camera 摄像头配置及状态模型
type Camera struct {
	ID               uint            `json:"id" gorm:"primaryKey"`
	Name             string          `json:"name" gorm:"size:128;not null"`          // 摄像头名称
	Brand            string          `json:"brand" gorm:"size:64;default:'hikvision'"` // 品牌: hikvision / general
	IP               string          `json:"ip" gorm:"size:64;not null"`             // 摄像头IP (支持跨网段)
	Subnet           string          `json:"subnet" gorm:"size:64"`                  // 所属网段标签，例如 192.168.2.0/24
	NetworkInterface string          `json:"network_interface" gorm:"size:32"`       // 绑定的出口网卡 (可选，如 eth0, eth1, 空则走系统路由)
	RTSPPort         int             `json:"rtsp_port" gorm:"default:554"`           // RTSP端口
	HTTPPort         int             `json:"http_port" gorm:"default:80"`            // HTTP/ONVIF端口
	Username         string          `json:"username" gorm:"size:64"`                // 访问账号
	PasswordEnc      string          `json:"-" gorm:"size:256"`                      // 敏感信息：加密保存的密码
	Group            string          `json:"group" gorm:"size:64;default:'默认分组'"`   // 分组 (楼层/区域)
	Channel          int             `json:"channel" gorm:"default:1"`               // 通道号 (默认通道1)
	
	// 码流配置
	MainStreamURL    string          `json:"main_stream_url" gorm:"size:512"`        // 主码流RTSP地址
	SubStreamURL     string          `json:"sub_stream_url" gorm:"size:512"`         // 子码流RTSP地址
	TransportMode    StreamTransport `json:"transport_mode" gorm:"default:'tcp'"`    // 传输模式: tcp/udp
	VideoCodec       string          `json:"video_codec" gorm:"size:32"`             // H.264 / H.265 / MPEG4
	Resolution       string          `json:"resolution" gorm:"size:32"`              // 分辨率，例如 1920x1080
	FrameRate        int             `json:"frame_rate"`                             // 帧率，如 25
	BitrateKbps      int             `json:"bitrate_kbps"`                           // 实时码率 Kbps

	// 设备元信息
	Model            string          `json:"model" gorm:"size:128"`                  // 设备型号
	SerialNumber     string          `json:"serial_number" gorm:"size:128"`          // 序列号
	FirmwareVersion  string          `json:"firmware_version" gorm:"size:128"`       // 固件版本

	// 独立录像配置 (第7节需求)
	RecordEnabled    bool            `json:"record_enabled" gorm:"default:false"`    // 独立录像开关
	RecordMode       RecordMode      `json:"record_mode" gorm:"default:'continuous'"` // 录像模式
	StorageID        uint            `json:"storage_id" gorm:"default:1"`            // 分配的存储位置ID (关联 Storage 表)
	RetentionDays    int             `json:"retention_days" gorm:"default:7"`        // 录像保留天数
	SegmentMinutes   int             `json:"segment_minutes" gorm:"default:5"`       // 录像单段时长(分钟)

	// 运行状态
	IsOnline         bool            `json:"is_online" gorm:"default:false"`         // 在线状态
	IsRecording      bool            `json:"is_recording" gorm:"default:false"`      // 当前实际录像状态 (REC标识来源)
	LastOnlineTime   *time.Time      `json:"last_online_time"`                       // 最近上线时间
	LastOfflineTime  *time.Time      `json:"last_offline_time"`                      // 最近离线时间
	LastOfflineReason string         `json:"last_offline_reason" gorm:"size:256"`    // 离线/异常原因

	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// CameraDTO 外部API数据传输对象，隐藏真实密码
type CameraDTO struct {
	Camera
	HasPassword bool `json:"has_password"` // 标记是否已配置密码
}
