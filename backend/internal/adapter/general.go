package adapter

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"istore-nvr/internal/model"
)

// GeneralAdapter 通用 RTSP / ONVIF / TP-Link / 雄迈 / 宇视 适配器
type GeneralAdapter struct{}

func NewGeneralAdapter() *GeneralAdapter {
	return &GeneralAdapter{}
}

// BuildStreamURLs 依据品牌模板或已有自定义配置生成码流
func (g *GeneralAdapter) BuildStreamURLs(camera *model.Camera, password string) (string, string) {
	// 若用户已显式配置了自定义流地址，优先保留
	if camera.MainStreamURL != "" && camera.SubStreamURL != "" {
		return camera.MainStreamURL, camera.SubStreamURL
	}

	rtspPort := camera.RTSPPort
	if rtspPort <= 0 {
		rtspPort = 554
	}

	userPass := ""
	if camera.Username != "" {
		encodedUser := url.QueryEscape(camera.Username)
		encodedPass := url.QueryEscape(password)
		userPass = fmt.Sprintf("%s:%s@", encodedUser, encodedPass)
	}

	portStr := ""
	if rtspPort != 554 {
		portStr = fmt.Sprintf(":%d", rtspPort)
	}

	brand := strings.ToLower(camera.Brand)
	var mainPath, subPath string

	switch brand {
	case "tplink", "mercury", "fast":
		mainPath = "/stream1"
		subPath = "/stream2"
	case "xiongmai", "xm":
		mainPath = "/live/ch0"
		subPath = "/live/ch1"
	case "uniview", "unv":
		mainPath = "/media/video1"
		subPath = "/media/video2"
	default:
		// 通用备用
		if camera.MainStreamURL != "" {
			return camera.MainStreamURL, camera.SubStreamURL
		}
		mainPath = "/live/ch0"
		subPath = "/live/ch1"
	}

	mainURL := fmt.Sprintf("rtsp://%s%s%s%s", userPass, camera.IP, portStr, mainPath)
	subURL := fmt.Sprintf("rtsp://%s%s%s%s", userPass, camera.IP, portStr, subPath)
	return mainURL, subURL
}

// ConnectTest 测试通用 RTSP / ONVIF 连通性
func (g *GeneralAdapter) ConnectTest(camera *model.Camera, password string) (*DeviceInfo, error) {
	timeout := 4 * time.Second
	rtspPort := camera.RTSPPort
	if rtspPort <= 0 {
		rtspPort = 554
	}

	addr := net.JoinHostPort(camera.IP, strconv.Itoa(rtspPort))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("设备 RTSP 端口 (%s) 无法连接: %w", addr, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	mainURL, subURL := g.BuildStreamURLs(camera, password)

	// 首先尝试 OPTIONS 请求
	optionsReq := fmt.Sprintf("OPTIONS %s RTSP/1.0\r\nCSeq: 1\r\nUser-Agent: iStore-NVR\r\n\r\n", mainURL)
	if _, err := conn.Write([]byte(optionsReq)); err != nil {
		return nil, fmt.Errorf("RTSP 报文发送失败: %w", err)
	}

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("RTSP 响应读取超时: %w", err)
	}

	if !strings.HasPrefix(statusLine, "RTSP/1.0 200") && !strings.HasPrefix(statusLine, "RTSP/1.0 401") {
		return nil, fmt.Errorf("RTSP 握手返回异常状态: %s", strings.TrimSpace(statusLine))
	}

	brandName := camera.Brand
	if brandName == "" {
		brandName = "general"
	}

	return &DeviceInfo{
		Brand:         brandName,
		Model:         fmt.Sprintf("网络摄像机 (%s)", brandName),
		RTSPPort:      rtspPort,
		MainStreamURL: mainURL,
		SubStreamURL:  subURL,
		IsOnline:      true,
		Resolution:    "1920x1080",
		VideoCodec:    "H.264",
	}, nil
}

func (g *GeneralAdapter) FetchDeviceInfo(camera *model.Camera, password string) (*DeviceInfo, error) {
	return g.ConnectTest(camera, password)
}
