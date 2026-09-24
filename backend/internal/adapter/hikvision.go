package adapter

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
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

// HikvisionAdapter 海康威视摄像头专用适配器
type HikvisionAdapter struct{}

func NewHikvisionAdapter() *HikvisionAdapter {
	return &HikvisionAdapter{}
}

// XML 结构体用于解析海康 ISAPI 响应
type HikDeviceInfoXML struct {
	XMLName         xml.Name `xml:"DeviceInfo"`
	DeviceName      string   `xml:"deviceName"`
	Model           string   `xml:"model"`
	SerialNumber    string   `xml:"serialNumber"`
	FirmwareVersion string   `xml:"firmwareVersion"`
	FirmwareReleasedDate string `xml:"firmwareReleasedDate"`
	EncoderVersion  string   `xml:"encoderVersion"`
}

type HikStreamingChannelListXML struct {
	XMLName  xml.Name            `xml:"StreamingChannelList"`
	Channels []HikStreamingChan `xml:"StreamingChannel"`
}

type HikStreamingChan struct {
	ID        string `xml:"id"`
	ChannelName string `xml:"channelName"`
	Enabled   bool   `xml:"enabled"`
	Video     struct {
		Enabled         bool   `xml:"enabled"`
		VideoCodecType  string `xml:"videoCodecType"`
		VideoResolutionWidth  int `xml:"videoResolutionWidth"`
		VideoResolutionHeight int `xml:"videoResolutionHeight"`
		MaxFrameRate    int    `xml:"maxFrameRate"`
	} `xml:"Video"`
}

// BuildStreamURLs 生成海康标准主码流与子码流地址
func (h *HikvisionAdapter) BuildStreamURLs(camera *model.Camera, password string) (string, string) {
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

	mainChanID := channel*100 + 1 // 如 101
	subChanID := channel*100 + 2  // 如 102

	mainURL := fmt.Sprintf("rtsp://%s%s:%d/Streaming/Channels/%d", userPass, camera.IP, rtspPort, mainChanID)
	subURL := fmt.Sprintf("rtsp://%s%s:%d/Streaming/Channels/%d", userPass, camera.IP, rtspPort, subChanID)

	return mainURL, subURL
}

// getDialer 根据网络接口配置生成跨网段拨号器
func (h *HikvisionAdapter) getDialer(netInterface string, timeout time.Duration) *net.Dialer {
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	if netInterface != "" {
		// 如果软路由指定了物理网卡接口，尝试获取该网卡的本地 IP 进行绑定
		if iface, err := net.InterfaceByName(netInterface); err == nil {
			if addrs, err := iface.Addrs(); err == nil {
				for _, addr := range addrs {
					if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
						if ipnet.IP.To4() != nil {
							dialer.LocalAddr = &net.TCPAddr{IP: ipnet.IP}
							break
						}
					}
				}
			}
		}
	}
	return dialer
}

// ConnectTest 跨网段快速连通性测试 (网络层与应用层探针)
func (h *HikvisionAdapter) ConnectTest(camera *model.Camera, password string) (*DeviceInfo, error) {
	timeout := 4 * time.Second
	dialer := h.getDialer(camera.NetworkInterface, timeout)

	// 1. 测试 RTSP 端口联通性
	rtspPort := camera.RTSPPort
	if rtspPort <= 0 {
		rtspPort = 554
	}
	rtspAddr := net.JoinHostPort(camera.IP, strconv.Itoa(rtspPort))
	rtspConn, err := dialer.Dial("tcp", rtspAddr)
	if err != nil {
		return nil, fmt.Errorf("RTSP 端口 (%s) 无法连接: %w，请检查跨网段路由或防火墙设置", rtspAddr, err)
	}
	_ = rtspConn.Close()

	// 2. 尝试通过 ISAPI 或 RTSP 握手获取详细信息
	info, fetchErr := h.FetchDeviceInfo(camera, password)
	if fetchErr == nil && info != nil {
		info.IsOnline = true
		return info, nil
	}

	// 3. 若 ISAPI 端口未开放或受限，通过原生 RTSP DESCRIBE 握手探针校验
	rtspInfo, rtspErr := h.probeRTSP(camera, password, dialer)
	if rtspErr == nil && rtspInfo != nil {
		rtspInfo.IsOnline = true
		return rtspInfo, nil
	}

	// 至少证明网络层端口可达
	mainStream, subStream := h.BuildStreamURLs(camera, password)
	return &DeviceInfo{
		Brand:         "hikvision",
		MainStreamURL: mainStream,
		SubStreamURL:  subStream,
		IsOnline:      true,
		Resolution:    "1920x1080 (推测)",
		VideoCodec:    "H.264",
	}, nil
}

// FetchDeviceInfo 通过海康 ISAPI 协议获取丰富元数据
func (h *HikvisionAdapter) FetchDeviceInfo(camera *model.Camera, password string) (*DeviceInfo, error) {
	httpPort := camera.HTTPPort
	if httpPort <= 0 {
		httpPort = 80
	}

	timeout := 4 * time.Second
	dialer := h.getDialer(camera.NetworkInterface, timeout)
	transport := &http.Transport{
		DialContext:       dialer.DialContext,
		DisableKeepAlives: true,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	apiURL := fmt.Sprintf("http://%s:%d/ISAPI/System/deviceInfo", camera.IP, httpPort)
	resp, body, err := h.doDigestOrBasicRequest(client, "GET", apiURL, camera.Username, password, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ISAPI 响应状态码异常: %d", resp.StatusCode)
	}

	var devXML HikDeviceInfoXML
	if err := xml.Unmarshal(body, &devXML); err != nil {
		return nil, fmt.Errorf("解析海康 ISAPI XML 失败: %w", err)
	}

	mainStream, subStream := h.BuildStreamURLs(camera, password)

	info := &DeviceInfo{
		Brand:           "hikvision",
		Model:           devXML.Model,
		SerialNumber:    devXML.SerialNumber,
		FirmwareVersion: devXML.FirmwareVersion,
		MainStreamURL:   mainStream,
		SubStreamURL:    subStream,
		IsOnline:        true,
	}

	// 尝试获取通道编码参数
	chanURL := fmt.Sprintf("http://%s:%d/ISAPI/Streaming/channels", camera.IP, httpPort)
	if chanResp, chanBody, err := h.doDigestOrBasicRequest(client, "GET", chanURL, camera.Username, password, nil); err == nil && chanResp.StatusCode == http.StatusOK {
		var chanList HikStreamingChannelListXML
		if xml.Unmarshal(chanBody, &chanList) == nil && len(chanList.Channels) > 0 {
			info.ChannelCount = len(chanList.Channels)
			for _, ch := range chanList.Channels {
				if ch.ID == "101" || ch.ID == "1" {
					info.VideoCodec = ch.Video.VideoCodecType
					if ch.Video.VideoResolutionWidth > 0 && ch.Video.VideoResolutionHeight > 0 {
						info.Resolution = fmt.Sprintf("%dx%d", ch.Video.VideoResolutionWidth, ch.Video.VideoResolutionHeight)
					}
					info.FrameRate = ch.Video.MaxFrameRate / 100
					if info.FrameRate == 0 {
						info.FrameRate = ch.Video.MaxFrameRate
					}
					break
				}
			}
		}
	}

	if info.VideoCodec == "" {
		info.VideoCodec = "H.264"
	}
	if info.Resolution == "" {
		info.Resolution = "1920x1080"
	}

	return info, nil
}

// probeRTSP 使用原生 RTSP TCP 报文发送 DESCRIBE 请求探测流媒体通道
func (h *HikvisionAdapter) probeRTSP(camera *model.Camera, password string, dialer *net.Dialer) (*DeviceInfo, error) {
	rtspPort := camera.RTSPPort
	if rtspPort <= 0 {
		rtspPort = 554
	}
	addr := net.JoinHostPort(camera.IP, strconv.Itoa(rtspPort))
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	mainStream, subStream := h.BuildStreamURLs(camera, password)
	// 发送 OPTIONS
	optionsReq := fmt.Sprintf("OPTIONS %s RTSP/1.0\r\nCSeq: 1\r\nUser-Agent: iStore-NVR\r\n\r\n", mainStream)
	if _, err := conn.Write([]byte(optionsReq)); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(line, "RTSP/1.0 200") && !strings.HasPrefix(line, "RTSP/1.0 401") {
		return nil, fmt.Errorf("RTSP 握手非预期响应: %s", strings.TrimSpace(line))
	}

	return &DeviceInfo{
		Brand:         "hikvision",
		MainStreamURL: mainStream,
		SubStreamURL:  subStream,
		IsOnline:      true,
		VideoCodec:    "H.264",
		Resolution:    "1920x1080",
	}, nil
}

// doDigestOrBasicRequest 处理 HTTP Digest 鉴权 (海康摄像头的标准认证方式)
func (h *HikvisionAdapter) doDigestOrBasicRequest(client *http.Client, method, targetURL, username, password string, body io.Reader) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		return nil, nil, err
	}

	// 首次发起请求获取 401 质询
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return resp, respBody, nil
	}

	// 解析 WWW-Authenticate 质询头
	authHeader := resp.Header.Get("WWW-Authenticate")
	_ = resp.Body.Close()

	if strings.HasPrefix(authHeader, "Digest ") {
		digestParts := parseDigestHeader(authHeader[7:])
		realm := digestParts["realm"]
		nonce := digestParts["nonce"]
		qop := digestParts["qop"]
		algorithm := digestParts["algorithm"]
		if algorithm == "" {
			algorithm = "MD5"
		}

		u, _ := url.Parse(targetURL)
		uri := u.RequestURI()
		nc := "00000001"
		cnonce := "0a4f113b"

		ha1 := md5Hex(fmt.Sprintf("%s:%s:%s", username, realm, password))
		ha2 := md5Hex(fmt.Sprintf("%s:%s", method, uri))

		var response string
		if strings.Contains(qop, "auth") {
			response = md5Hex(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, "auth", ha2))
		} else {
			response = md5Hex(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))
		}

		authVal := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s", algorithm="%s"`,
			username, realm, nonce, uri, response, algorithm)
		if strings.Contains(qop, "auth") {
			authVal += fmt.Sprintf(`, qop=auth, nc=%s, cnonce="%s"`, nc, cnonce)
		}

		authReq, _ := http.NewRequest(method, targetURL, body)
		authReq.Header.Set("Authorization", authVal)
		authResp, err := client.Do(authReq)
		if err != nil {
			return nil, nil, err
		}
		respBody, _ := io.ReadAll(authResp.Body)
		_ = authResp.Body.Close()
		return authResp, respBody, nil
	}

	// 回退至 Basic 认证
	basicReq, _ := http.NewRequest(method, targetURL, body)
	basicReq.SetBasicAuth(username, password)
	authResp, err := client.Do(basicReq)
	if err != nil {
		return nil, nil, err
	}
	respBody, _ := io.ReadAll(authResp.Body)
	_ = authResp.Body.Close()
	return authResp, respBody, nil
}

func parseDigestHeader(header string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(header, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
			result[key] = val
		}
	}
	return result
}

func md5Hex(s string) string {
	h := md5.New()
	_, _ = h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
