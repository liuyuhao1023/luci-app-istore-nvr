package adapter

import "istore-nvr/internal/model"

// DeviceInfo 摄像头能力与元信息
type DeviceInfo struct {
	Brand           string `json:"brand"`
	Model           string `json:"model"`
	SerialNumber    string `json:"serial_number"`
	FirmwareVersion string `json:"firmware_version"`
	ChannelCount    int    `json:"channel_count"`
	VideoCodec      string `json:"video_codec"`   // H.264, H.265, etc.
	Resolution      string `json:"resolution"`    // 1920x1080
	FrameRate       int    `json:"frame_rate"`
	MainStreamURL   string `json:"main_stream_url"`
	SubStreamURL    string `json:"sub_stream_url"`
	IsOnline        bool   `json:"is_online"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

// DiscoveredDevice 扫描/探测发现到的设备信息
type DiscoveredDevice struct {
	IP           string `json:"ip"`
	Port         int    `json:"port"`
	Brand        string `json:"brand"` // hikvision / onvif / unknown
	Model        string `json:"model"`
	SerialNumber string `json:"serial_number"`
	Firmware     string `json:"firmware"`
	MAC          string `json:"mac"`
	Subnet       string `json:"subnet"`
	ProbeMethod  string `json:"probe_method"` // onvif_multicast, unicast_probe, port_scan
}

// CameraAdapter 统一的摄像头适配器接口
type CameraAdapter interface {
	// ConnectTest 测试跨网段/本网段摄像头的网络与认证连通性
	ConnectTest(camera *model.Camera, password string) (*DeviceInfo, error)

	// FetchDeviceInfo 从摄像头读取具体配置、编码与型号信息
	FetchDeviceInfo(camera *model.Camera, password string) (*DeviceInfo, error)

	// BuildStreamURLs 生成主码流与子码流的规范 RTSP 播放地址
	BuildStreamURLs(camera *model.Camera, password string) (mainStream, subStream string)
}
