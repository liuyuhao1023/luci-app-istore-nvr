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

// DahuaAdapter 大华 (Dahua / 乐橙 / 丰润) IPC 适配器
type DahuaAdapter struct{}

func NewDahuaAdapter() *DahuaAdapter {
	return &DahuaAdapter{}
}

// BuildStreamURLs 生成大华标准主码流与子码流地址
func (d *DahuaAdapter) BuildStreamURLs(camera *model.Camera, password string) (string, string) {
	channel := camera.Channel
	if channel <= 0 {
		channel = 1
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

	// 大华标准取流规范：/cam/realmonitor?channel=1&subtype=0 (主), subtype=1 (子)
	mainURL := fmt.Sprintf("rtsp://%s%s:%d/cam/realmonitor?channel=%d&subtype=0", userPass, camera.IP, rtspPort, channel)
	subURL := fmt.Sprintf("rtsp://%s%s:%d/cam/realmonitor?channel=%d&subtype=1", userPass, camera.IP, rtspPort, channel)
	return mainURL, subURL
}

// ConnectTest 测试大华摄像头的连通性
func (d *DahuaAdapter) ConnectTest(camera *model.Camera, password string) (*DeviceInfo, error) {
	timeout := 4 * time.Second
	rtspPort := camera.RTSPPort
	if rtspPort <= 0 {
		rtspPort = 554
	}

	addr := net.JoinHostPort(camera.IP, strconv.Itoa(rtspPort))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("大华设备 RTSP 端口 (%s) 无法连接: %w", addr, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	mainURL, subURL := d.BuildStreamURLs(camera, password)
	describeReq := fmt.Sprintf("DESCRIBE %s RTSP/1.0\r\nCSeq: 1\r\nUser-Agent: iStore-NVR\r\nAccept: application/sdp\r\n\r\n", mainURL)
	if _, err := conn.Write([]byte(describeReq)); err != nil {
		return nil, fmt.Errorf("RTSP 报文发送失败: %w", err)
	}

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("RTSP 响应读取超时: %w", err)
	}

	if !strings.HasPrefix(statusLine, "RTSP/1.0 200") && !strings.HasPrefix(statusLine, "RTSP/1.0 401") {
		return nil, fmt.Errorf("大华 RTSP 握手返回异常状态: %s", strings.TrimSpace(statusLine))
	}

	return &DeviceInfo{
		Brand:         "dahua",
		Model:         "大华网络摄像机",
		RTSPPort:      rtspPort,
		MainStreamURL: mainURL,
		SubStreamURL:  subURL,
		IsOnline:      true,
		Resolution:    "1920x1080",
		VideoCodec:    "H.264",
	}, nil
}

func (d *DahuaAdapter) FetchDeviceInfo(camera *model.Camera, password string) (*DeviceInfo, error) {
	return d.ConnectTest(camera, password)
}
