package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"istore-nvr/internal/service"
)

// SetupRouter 初始化 Gin HTTP 路由
func SetupRouter(
	db *gorm.DB,
	cs *service.CameraService,
	ss *service.StorageService,
	re *service.RecordEngine,
	se *service.StreamEngine,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 跨域支持，支持前后端分离开发调试
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(corsConfig))

	apiGroup := r.Group("/api")
	{
		// 1. 摄像头管理路由
		camHandler := NewCameraHandler(cs, re)
		apiGroup.GET("/cameras", camHandler.ListCameras)
		apiGroup.POST("/cameras", camHandler.AddCamera)
		apiGroup.GET("/cameras/:id", camHandler.GetCamera)
		apiGroup.PUT("/cameras/:id", camHandler.UpdateCamera)
		apiGroup.DELETE("/cameras/:id", camHandler.DeleteCamera)
		apiGroup.POST("/cameras/test", camHandler.TestConnection)
		apiGroup.POST("/cameras/scan", camHandler.ScanDevices)
		apiGroup.PUT("/cameras/:id/record", camHandler.ToggleCameraRecord)
		apiGroup.POST("/cameras/batch/record", camHandler.BatchRecord)
		apiGroup.POST("/cameras/batch/storage", camHandler.BatchStorage)

		// 2. 存储管理路由 (含 SMB/CIFS 网络存储)
		storageHandler := NewStorageHandler(ss)
		apiGroup.GET("/storages", storageHandler.ListStorages)
		apiGroup.POST("/storages", storageHandler.AddStorage)
		apiGroup.PUT("/storages/:id", storageHandler.UpdateStorage)
		apiGroup.DELETE("/storages/:id", storageHandler.DeleteStorage)
		apiGroup.POST("/storages/:id/mount", storageHandler.MountSMB)

		// 3. 录像检索与全局开关
		recHandler := NewRecordHandler(db, re)
		apiGroup.GET("/record/global", recHandler.GetGlobalRecordSwitch)
		apiGroup.POST("/record/global", recHandler.SetGlobalRecordSwitch)
		apiGroup.GET("/record/query", recHandler.QueryRecordings)
		apiGroup.GET("/record/download/:id", recHandler.DownloadRecording)

		// 4. 实时流代理请求
		streamHandler := NewStreamHandler(se)
		apiGroup.GET("/stream/:id/request", streamHandler.RequestStream)
		apiGroup.POST("/stream/:id/release", streamHandler.ReleaseStream)

		// 5. 系统状态与真实监控指标
		sysHandler := NewSystemHandler(db, re, se)
		apiGroup.GET("/system/stats", sysHandler.GetStats)
	}

	// 托管前端静态资源 (优先检查当前工作目录或可执行文件同级目录下的 dist)
	distPath := "./dist"
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		if exe, err := os.Executable(); err == nil {
			cand := filepath.Join(filepath.Dir(exe), "dist")
			if _, err := os.Stat(cand); err == nil {
				distPath = cand
			}
		}
	}

	if _, err := os.Stat(distPath); err == nil {
		r.Static("/assets", filepath.Join(distPath, "assets"))
		r.StaticFile("/favicon.ico", filepath.Join(distPath, "favicon.ico"))
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(distPath, "index.html"))
		})
	} else {
		r.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service": "OpenWrt NVR Backend API",
				"status":  "running",
				"version": "1.0.0",
			})
		})
	}

	return r
}
