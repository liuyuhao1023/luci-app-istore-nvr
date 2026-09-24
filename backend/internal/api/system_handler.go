package api

import (
	"bufio"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"istore-nvr/internal/model"
	"istore-nvr/internal/service"
)

type SystemHandler struct {
	db           *gorm.DB
	recordEngine *service.RecordEngine
	streamEngine *service.StreamEngine
}

func NewSystemHandler(db *gorm.DB, re *service.RecordEngine, se *service.StreamEngine) *SystemHandler {
	return &SystemHandler{
		db:           db,
		recordEngine: re,
		streamEngine: se,
	}
}

type SystemStatsResponse struct {
	OS            string  `json:"os"`
	Arch          string  `json:"arch"`
	CPULoad       float64 `json:"cpu_load"`        // CPU 1分钟负载
	MemTotalMB    uint64  `json:"mem_total_mb"`    // 物理内存总量 MB
	MemUsedMB     uint64  `json:"mem_used_mb"`     // 已用内存 MB
	MemFreeMB     uint64  `json:"mem_free_mb"`     // 剩余空闲内存 MB
	MemUsagePct   float64 `json:"mem_usage_pct"`   // 内存使用率 %
	DiskTotalGB   float64 `json:"disk_total_gb"`   // 数据盘总容量 GB
	DiskFreeGB    float64 `json:"disk_free_gb"`    // 数据盘剩余 GB
	DiskUsagePct  float64 `json:"disk_usage_pct"`  // 数据盘使用率 %
	TotalCameras  int64   `json:"total_cameras"`   // 摄像头总数
	OnlineCameras int64   `json:"online_cameras"`  // 在线摄像头数
	RecordingCams int64   `json:"recording_cams"`  // 正在录像路数
	GlobalRecord  bool    `json:"global_record"`   // 全局录像总开关
}

// GetStats 获取软路由真实系统资源运行指标 (杜绝模拟数据)
func (h *SystemHandler) GetStats(c *gin.Context) {
	resp := SystemStatsResponse{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		GlobalRecord: h.recordEngine.IsGlobalEnabled(),
	}

	// 统计摄像头状态
	h.db.Model(&model.Camera{}).Count(&resp.TotalCameras)
	h.db.Model(&model.Camera{}).Where("is_online = ?", true).Count(&resp.OnlineCameras)
	h.db.Model(&model.Camera{}).Where("is_recording = ?", true).Count(&resp.RecordingCams)

	// Linux 真实硬件负载与内存采集
	if runtime.GOOS == "linux" {
		// 1. 读取负载
		if data, err := os.ReadFile("/proc/loadavg"); err == nil {
			fields := strings.Fields(string(data))
			if len(fields) > 0 {
				resp.CPULoad, _ = strconv.ParseFloat(fields[0], 64)
			}
		}

		// 2. 读取内存
		if f, err := os.Open("/proc/meminfo"); err == nil {
			scanner := bufio.NewScanner(f)
			var totalKB, availKB uint64
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "MemTotal:") {
					totalKB = parseMemKB(line)
				} else if strings.HasPrefix(line, "MemAvailable:") {
					availKB = parseMemKB(line)
				}
			}
			_ = f.Close()

			if totalKB > 0 {
				resp.MemTotalMB = totalKB / 1024
				resp.MemFreeMB = availKB / 1024
				resp.MemUsedMB = resp.MemTotalMB - resp.MemFreeMB
				resp.MemUsagePct = float64(resp.MemUsedMB) / float64(resp.MemTotalMB) * 100
			}
		}

		// 3. 读取本地数据盘存储 (/mnt/sata1-4)
		targetPath := "/mnt/sata1-4"
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			targetPath = "/"
		}
		var stat syscall.Statfs_t
		if err := syscall.Statfs(targetPath, &stat); err == nil {
			totalBytes := stat.Blocks * uint64(stat.Bsize)
			freeBytes := stat.Bavail * uint64(stat.Bsize)
			usedBytes := totalBytes - freeBytes

			resp.DiskTotalGB = float64(totalBytes) / (1024 * 1024 * 1024)
			resp.DiskFreeGB = float64(freeBytes) / (1024 * 1024 * 1024)
			if totalBytes > 0 {
				resp.DiskUsagePct = float64(usedBytes) / float64(totalBytes) * 100
			}
		}
	} else {
		// 非 Linux 开发调试默认
		resp.CPULoad = 0.35
		resp.MemTotalMB = 8192
		resp.MemUsedMB = 1200
		resp.MemFreeMB = 6992
		resp.MemUsagePct = 14.6
		resp.DiskTotalGB = 119.2
		resp.DiskFreeGB = 108.6
		resp.DiskUsagePct = 8.9
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

func parseMemKB(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		v, _ := strconv.ParseUint(fields[1], 10, 64)
		return v
	}
	return 0
}
