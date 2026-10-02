package adapter

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DetectBrand 自动探测局域网/跨网段摄像头的品牌厂商
// 支持主流品牌特征匹配：海康威视、中维世纪、大华股份、TP-Link、雄迈技术、宇视科技等
func DetectBrand(ip string, httpPort, rtspPort int) string {
	if httpPort <= 0 {
		httpPort = 80
	}
	if rtspPort <= 0 {
		rtspPort = 554
	}

	timeout := 1500 * time.Millisecond

	// 1. 优先通过 HTTP 响应头及特征路径探测
	httpBrand := detectBrandViaHTTP(ip, httpPort, timeout)
	if httpBrand != "" && httpBrand != "unknown" {
		return httpBrand
	}

	// 2. 探测专用特征端口
	// 海康 ISAPI/SDK 专有服务端口
	if isTCPPortOpen(ip, 8000, 600*time.Millisecond) {
		return "hikvision"
	}
	// 中维世纪 CloudSEE 视频通信端口
	if isTCPPortOpen(ip, 9101, 600*time.Millisecond) || isTCPPortOpen(ip, 9102, 600*time.Millisecond) {
		return "jovision"
	}
	// 大华专用私有服务端口
	if isTCPPortOpen(ip, 37777, 600*time.Millisecond) {
		return "dahua"
	}
	// 雄迈 (XM/NetSurveillance) 私有控制端口
	if isTCPPortOpen(ip, 34567, 600*time.Millisecond) {
		return "xiongmai"
	}

	// 3. 通过 RTSP OPTIONS/DESCRIBE 握手响应头检测
	rtspBrand := detectBrandViaRTSP(ip, rtspPort, timeout)
	if rtspBrand != "" && rtspBrand != "unknown" {
		return rtspBrand
	}

	return "general"
}

// detectBrandViaHTTP 探测 HTTP 端口 Server 头及特征响应
func detectBrandViaHTTP(ip string, port int, timeout time.Duration) string {
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	url := fmt.Sprintf("http://%s:%d/", ip, port)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "iStore-NVR")

	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		server := strings.ToLower(resp.Header.Get("Server"))

		// 中维世纪常见嵌入式 Web 服务器
		if strings.Contains(server, "thttpd") || strings.Contains(server, "jovision") {
			return "jovision"
		}
		// 海康威视
		if strings.Contains(server, "hikvision") || strings.Contains(server, "hikvision-webs") {
			return "hikvision"
		}
		// 大华
		if strings.Contains(server, "dahua") || strings.Contains(server, "uc-httpd") {
			return "dahua"
		}
		// TP-Link
		if strings.Contains(server, "tp-link") || strings.Contains(server, "mercury") {
			return "tplink"
		}
	}

	// 尝试中维世纪官方 CGI 探针
	jvsURL := fmt.Sprintf("http://%s:%d/cgi-bin/jvsweb.cgi?cmd=yst&action=get_video&username=admin&password=", ip, port)
	if jvsResp, jvsErr := client.Get(jvsURL); jvsErr == nil {
		defer jvsResp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(jvsResp.Body, 512))
		if strings.Contains(string(body), `"status":"ok"`) || strings.Contains(string(body), `live0.264`) {
			return "jovision"
		}
	}

	// 尝试海康 ISAPI 探针
	hikURL := fmt.Sprintf("http://%s:%d/ISAPI/System/deviceInfo", ip, port)
	if hikResp, hikErr := client.Get(hikURL); hikErr == nil {
		defer hikResp.Body.Close()
		if hikResp.StatusCode == http.StatusOK || hikResp.StatusCode == http.StatusUnauthorized {
			authHeader := strings.ToLower(hikResp.Header.Get("WWW-Authenticate"))
			if strings.Contains(authHeader, "digest") || hikResp.StatusCode == http.StatusOK {
				return "hikvision"
			}
		}
	}

	return ""
}

// detectBrandViaRTSP 通过标准 RTSP OPTIONS 获取流媒体服务器特征
func detectBrandViaRTSP(ip string, port int, timeout time.Duration) string {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, strconv.Itoa(port)), timeout)
	if err != nil {
		return ""
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	req := fmt.Sprintf("OPTIONS rtsp://%s:%d/ RTSP/1.0\r\nCSeq: 1\r\nUser-Agent: iStore-NVR\r\n\r\n", ip, port)
	if _, err := conn.Write([]byte(req)); err != nil {
		return ""
	}

	reader := bufio.NewReader(conn)
	for i := 0; i < 20; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}

		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "server:") {
			if strings.Contains(lower, "hikvision") {
				return "hikvision"
			}
			if strings.Contains(lower, "dahua") {
				return "dahua"
			}
			if strings.Contains(lower, "jovision") || strings.Contains(lower, "cloudsee") {
				return "jovision"
			}
			if strings.Contains(lower, "uniview") {
				return "uniview"
			}
		}
	}

	return ""
}
