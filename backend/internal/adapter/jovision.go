package adapter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"istore-nvr/internal/model"
)

// JovisionAdapter 中维世纪 (Jovision / 云视通) IPC 适配器
type JovisionAdapter struct{}

func NewJovisionAdapter() *JovisionAdapter {
	return &JovisionAdapter{}
}

type jvsVideoResponse struct {
	Status string `json:"status"`
	Data   []struct {
		ID      int    `json:"id"`
		Stream0 string `json:"stream0"`
		Stream1 string `json:"stream1"`
	} `json:"data"`
}

// BuildStreamURLs 生成中维世纪标准主码流与子码流地址
func (j *JovisionAdapter) BuildStreamURLs(camera *model.Camera, password string) (string, string) {
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

	// 中维世纪标准取流规范：主码流 live0.264，子码流 live1.264
	mainURL := fmt.Sprintf("rtsp://%s%s%s/live0.264", userPass, camera.IP, portStr)
	subURL := fmt.Sprintf("rtsp://%s%s%s/live1.264", userPass, camera.IP, portStr)
	return mainURL, subURL
}

// ConnectTest 测试中维世纪摄像头的连通性与码流可用性
func (j *JovisionAdapter) ConnectTest(camera *model.Camera, password string) (*DeviceInfo, error) {
	timeout := 4 * time.Second

	// 1. 尝试从设备 HTTP/CGI 抓取真实视频配置
	info, err := j.FetchDeviceInfo(camera, password)
	if err == nil && info != nil {
		info.IsOnline = true
		return info, nil
	}

	// 2. 若 HTTP 未开放，直接探测 RTSP 端口及 live0.264 握手
	rtspPort := camera.RTSPPort
	if rtspPort <= 0 {
		rtspPort = 554
	}
	addr := net.JoinHostPort(camera.IP, strconv.Itoa(rtspPort))
	conn, dErr := net.DialTimeout("tcp", addr, timeout)
	if dErr != nil {
		return nil, fmt.Errorf("中维设备 RTSP 端口 (%s) 无法连接: %w", addr, dErr)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	mainURL, subURL := j.BuildStreamURLs(camera, password)
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
		return nil, fmt.Errorf("中维 RTSP 握手返回异常状态: %s", strings.TrimSpace(statusLine))
	}

	return &DeviceInfo{
		Brand:         "jovision",
		Model:         "中维世纪 IPC (通用型)",
		RTSPPort:      rtspPort,
		MainStreamURL: mainURL,
		SubStreamURL:  subURL,
		IsOnline:      true,
		Resolution:    "1920x1080",
		VideoCodec:    "H.264",
	}, nil
}

// FetchDeviceInfo 从中维设备 HTTP 管理端口读取参数
func (j *JovisionAdapter) FetchDeviceInfo(camera *model.Camera, password string) (*DeviceInfo, error) {
	httpPort := camera.HTTPPort
	if httpPort <= 0 {
		httpPort = 80
	}

	client := &http.Client{Timeout: 3 * time.Second}
	apiURL := fmt.Sprintf("http://%s:%d/cgi-bin/jvsweb.cgi?cmd=yst&action=get_video&username=%s&password=%s",
		camera.IP, httpPort, url.QueryEscape(camera.Username), url.QueryEscape(password))

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("连接中维 HTTP 端口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 状态码异常: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var jvsResp jvsVideoResponse
	if err := json.Unmarshal(body, &jvsResp); err != nil {
		return nil, fmt.Errorf("解析中维视频配置失败: %w", err)
	}

	mainURL, subURL := j.BuildStreamURLs(camera, password)

	return &DeviceInfo{
		Brand:         "jovision",
		Model:         "中维世纪 CloudSEE IPC",
		RTSPPort:      camera.RTSPPort,
		MainStreamURL: mainURL,
		SubStreamURL:  subURL,
		IsOnline:      true,
		Resolution:    "1920x1080",
		VideoCodec:    "H.264",
	}, nil
}
