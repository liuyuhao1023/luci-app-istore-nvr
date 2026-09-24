package model

import "time"

// RecordingIndex 录像片段索引模型
type RecordingIndex struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	CameraID    uint      `json:"camera_id" gorm:"index;not null"`             // 对应摄像头ID
	StorageID   uint      `json:"storage_id" gorm:"index;not null"`            // 对应存储设备ID
	FilePath    string    `json:"file_path" gorm:"size:512;not null;uniqueIndex"` // 文件相对存储根目录的路径，例如: recordings/1/2026/09/24/rec_1_20260924_180000.mp4
	Format      string    `json:"format" gorm:"size:16;default:'mp4'"`         // mp4 / ts
	VideoCodec  string    `json:"video_codec" gorm:"size:32"`                  // H.264 / H.265
	StartTime   time.Time `json:"start_time" gorm:"index;not null"`            // 片段开始时间
	EndTime     time.Time `json:"end_time" gorm:"index;not null"`              // 片段结束时间
	DurationSec int       `json:"duration_sec"`                                // 实际录像时长 (秒)
	SizeBytes   int64     `json:"size_bytes"`                                  // 文件字节大小
	IsProtected bool      `json:"is_protected" gorm:"default:false"`           // 是否防清理锁定
	CreatedAt   time.Time `json:"created_at"`
}

// SystemConfig 系统全局配置
type SystemConfig struct {
	ID                  uint   `json:"id" gorm:"primaryKey"`
	GlobalRecordEnabled bool   `json:"global_record_enabled" gorm:"default:true"` // 全局录像功能总开关
	ServerPort          int    `json:"server_port" gorm:"default:8080"`           // Web管理后台HTTP端口
	RTSPListenPort      int    `json:"rtsp_listen_port" gorm:"default:8554"`      // 流代理RTSP端口
	WebRTCPort          int    `json:"webrtc_port" gorm:"default:8889"`           // WebRTC播放端口
	AutoCleanCron       string `json:"auto_clean_cron" gorm:"default:'0 3 * * *'"` // 存储清理执行周期
}

// AlertEvent 系统告警与历史事件
type AlertEvent struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Level     string    `json:"level" gorm:"size:16"`    // info, warning, error, critical
	Source    string    `json:"source" gorm:"size:64"`   // camera, storage, system, record
	SourceID  uint      `json:"source_id"`               // 关联源对象ID
	Message   string    `json:"message" gorm:"size:512"` // 告警消息
	CreatedAt time.Time `json:"created_at" gorm:"index"`
}
