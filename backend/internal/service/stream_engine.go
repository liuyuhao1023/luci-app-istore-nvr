package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"istore-nvr/internal/model"
	"istore-nvr/internal/utils"
)

// StreamEngine 流媒体网关服务 (深度集成 MediaMTX 实现超低延迟 WebRTC/HLS 播放与流复用)
type StreamEngine struct {
	db          *gorm.DB
	mediaMTXCmd *exec.Cmd
	mu          sync.Mutex
	registered  map[string]bool
	httpClient  *http.Client
}

func NewStreamEngine(db *gorm.DB) *StreamEngine {
	engine := &StreamEngine{
		db:         db,
		registered: make(map[string]bool),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	// 启动并守护 MediaMTX 进程
	go engine.ensureMediaMTX()

	return engine
}

// Close 停止并清理流媒体子进程
func (e *StreamEngine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.mediaMTXCmd != nil && e.mediaMTXCmd.Process != nil {
		_ = e.mediaMTXCmd.Process.Kill()
	}
	_ = os.Remove("/var/run/mediamtx.pid")
}

// ensureMediaMTX 确保 MediaMTX 正在后台运行
func (e *StreamEngine) ensureMediaMTX() {
	if runtime.GOOS != "linux" {
		log.Println("[StreamEngine] 非 Linux 平台，跳过本地 MediaMTX 启动")
		return
	}

	// 动态查找 mediamtx 可执行文件：依次搜索 PATH、当前程序目录、/usr/bin
	binPath := ""
	if p, err := exec.LookPath("mediamtx"); err == nil {
		binPath = p
	} else if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "mediamtx")
		if _, err := os.Stat(candidate); err == nil {
			binPath = candidate
		}
	}
	if binPath == "" {
		if _, err := os.Stat("/usr/bin/mediamtx"); err == nil {
			binPath = "/usr/bin/mediamtx"
		}
	}

	if binPath == "" {
		log.Println("[StreamEngine] 警告: 未找到 mediamtx 可执行文件，流媒体服务无法启动")
		return
	}

	confPath := "/tmp/mediamtx.yml"

	// 动态检测 OpenWrt 局域网 IP
	lanIP := "192.168.1.1"
	if out, err := exec.Command("uci", "-q", "get", "network.lan.ipaddr").Output(); err == nil {
		t := strings.TrimSpace(string(out))
		if t != "" {
			lanIP = t
		}
	}

	// 生成基础配置 (开启 API, WebRTC, HLS, RTSP)
	confContent := fmt.Sprintf(`logLevel: info
logDestinations: [stdout]

api: yes
apiAddress: 127.0.0.1:9997

rtsp: yes
rtspAddress: :8554
protocols: [tcp]

webrtc: yes
webrtcAddress: :8889
webrtcEncryption: no
webrtcIPsFromInterfaces: no
webrtcAdditionalHosts:
  - %s

hls: yes
hlsAddress: :8888

rtmp: no
srt: no

paths:
  all:
    name: ~^.*$
`, lanIP)
	_ = os.WriteFile(confPath, []byte(confContent), 0644)

	// 若存在旧 pid，安全清理该进程，严禁使用 killall 影响容器
	if pidBytes, err := os.ReadFile("/var/run/mediamtx.pid"); err == nil {
		if oldPid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes))); err == nil && oldPid > 0 {
			if proc, err := os.FindProcess(oldPid); err == nil {
				_ = proc.Kill()
			}
		}
		_ = os.Remove("/var/run/mediamtx.pid")
		time.Sleep(200 * time.Millisecond)
	}

	log.Printf("[StreamEngine] 正在启动 MediaMTX: %s", binPath)
	cmd := exec.Command(binPath, confPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Printf("[StreamEngine] 启动 MediaMTX 失败: %v", err)
		return
	}

	if cmd.Process != nil {
		_ = os.WriteFile("/var/run/mediamtx.pid", []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
	}

	e.mu.Lock()
	e.mediaMTXCmd = cmd
	e.mu.Unlock()

	// 等待并同步当前数据库已有的所有在线摄像头
	time.Sleep(2 * time.Second)
	e.syncAllCamerasToMediaMTX()
}

// MediaMTXPathConfig 对应 MediaMTX REST API 的 Path 格式
type MediaMTXPathConfig struct {
	Source         string `json:"source"`
	SourceOnDemand bool   `json:"sourceOnDemand"`
	RTSPTransport  string `json:"rtspTransport"`
}

// RequestStream 请求单路摄像头实时播放流信息
func (e *StreamEngine) RequestStream(cameraID uint, streamType string, clientHost string) (map[string]interface{}, error) {
	if streamType != "main" && streamType != "sub" {
		streamType = "sub" // 默认子码流，保护性能
	}

	var cam model.Camera
	if err := e.db.First(&cam, cameraID).Error; err != nil {
		return nil, fmt.Errorf("摄像头 ID=%d 不存在", cameraID)
	}

	rawPass, _ := utils.DecryptPassword(cam.PasswordEnc)
	var rtspURL string
	if streamType == "main" {
		rtspURL = cam.MainStreamURL
	} else {
		rtspURL = cam.SubStreamURL
	}

	if rtspURL == "" {
		chanNum := 101
		if streamType == "sub" {
			chanNum = 102
		}
		rtspURL = fmt.Sprintf("rtsp://%s:%s@%s:%d/Streaming/Channels/%d",
			cam.Username, rawPass, cam.IP, cam.RTSPPort, chanNum)
	}

	pathName := fmt.Sprintf("cam_%d_%s", cameraID, streamType)

	// 注册到 MediaMTX
	if err := e.registerPathToMediaMTX(pathName, rtspURL); err != nil {
		log.Printf("[StreamEngine] 注册流路径 %s 失败: %v", pathName, err)
	}

	if clientHost == "" {
		clientHost = "192.168.1.15"
	}

	// 返回前端 WebRTC 与 HLS 播放信息
	return map[string]interface{}{
		"camera_id":   cameraID,
		"stream_type": streamType,
		"path_name":   pathName,
		"whep_url":    fmt.Sprintf("http://%s:8889/%s/whep", clientHost, pathName),
		"hls_url":     fmt.Sprintf("http://%s:8888/%s/index.m3u8", clientHost, pathName),
		"iframe_url":  fmt.Sprintf("http://%s:8889/%s/?autoplay=true&muted=true&controls=false", clientHost, pathName),
	}, nil
}

// ReleaseStream 客户端关闭画面时释放引用
func (e *StreamEngine) ReleaseStream(cameraID uint, streamType string) {
	// MediaMTX 支持 sourceOnDemand，无客户端请求时自动断开拉流
}

// registerPathToMediaMTX 向 MediaMTX API 动态添加流路径
func (e *StreamEngine) registerPathToMediaMTX(pathName, rtspSource string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	config := MediaMTXPathConfig{
		Source:         rtspSource,
		SourceOnDemand: true,
		RTSPTransport:  "tcp",
	}

	bodyBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}

	// 尝试 add
	apiURL := fmt.Sprintf("http://127.0.0.1:9997/v3/config/paths/add/%s", pathName)
	resp, err := e.httpClient.Post(apiURL, "application/json", bytes.NewReader(bodyBytes))
	if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated) {
		_ = resp.Body.Close()
		e.registered[pathName] = true
		log.Printf("[StreamEngine] 成功注册流媒体路径: %s -> %s", pathName, rtspSource)
		return nil
	}

	if resp != nil {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		// 如果已存在，尝试 patch 更新
		if resp.StatusCode == http.StatusBadRequest && bytes.Contains(respBody, []byte("already")) {
			patchURL := fmt.Sprintf("http://127.0.0.1:9997/v3/config/paths/patch/%s", pathName)
			req, reqErr := http.NewRequest(http.MethodPatch, patchURL, bytes.NewReader(bodyBytes))
			if reqErr == nil {
				req.Header.Set("Content-Type", "application/json")
				if pResp, pErr := e.httpClient.Do(req); pErr == nil {
					_ = pResp.Body.Close()
					e.registered[pathName] = true
					return nil
				}
			}
		}
		return fmt.Errorf("API 响应码: %d, 内容: %s", resp.StatusCode, string(respBody))
	}

	return err
}

// syncAllCamerasToMediaMTX 将当前数据库中所有摄像头预注册到 MediaMTX
func (e *StreamEngine) syncAllCamerasToMediaMTX() {
	var cameras []model.Camera
	if err := e.db.Find(&cameras).Error; err != nil {
		return
	}

	for _, cam := range cameras {
		rawPass, _ := utils.DecryptPassword(cam.PasswordEnc)
		mainURL := cam.MainStreamURL
		subURL := cam.SubStreamURL
		if mainURL == "" {
			mainURL = fmt.Sprintf("rtsp://%s:%s@%s:%d/Streaming/Channels/101",
				cam.Username, rawPass, cam.IP, cam.RTSPPort)
		}
		if subURL == "" {
			subURL = fmt.Sprintf("rtsp://%s:%s@%s:%d/Streaming/Channels/102",
				cam.Username, rawPass, cam.IP, cam.RTSPPort)
		}

		_ = e.registerPathToMediaMTX(fmt.Sprintf("cam_%d_main", cam.ID), mainURL)
		_ = e.registerPathToMediaMTX(fmt.Sprintf("cam_%d_sub", cam.ID), subURL)
	}
}
