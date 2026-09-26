package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"istore-nvr/internal/api"
	"istore-nvr/internal/repository"
	"istore-nvr/internal/service"
)

func main() {
	var (
		portFlag = flag.Int("port", 8080, "Web 管理后台服务监听端口")
		dbFlag   = flag.String("db", "", "SQLite 数据库存储路径 (默认自动选择安全数据盘)")
	)
	flag.Parse()

	log.Println("==================================================")
	log.Println("  OpenWrt 网络视频监控管理系统 (NVR 摄像头管理) v1.0 ")
	log.Println("==================================================")

	// 1. 确定安全的数据库落盘路径 (绝对禁止在软路由 /overlay 上持续写库)
	dbPath := *dbFlag
	if dbPath == "" {
		dbPath = filepath.Join("data", "nvr.db")
	}
	log.Printf("[Init] 数据库存储路径: %s\n", dbPath)

	// 2. 初始化持久层与数据表结构
	db, err := repository.InitDB(dbPath)
	if err != nil {
		log.Fatalf("[Fatal] 数据库初始化失败: %v\n", err)
	}

	// 3. 初始化核心业务服务
	storageService := service.NewStorageService(db)
	cameraService := service.NewCameraService(db)
	recordEngine := service.NewRecordEngine(db, storageService, cameraService)
	streamEngine := service.NewStreamEngine(db)

	// 4. 启动后台摄像头健康心跳协程池 (每 30 秒并发轮询)
	heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())
	defer heartbeatCancel()
	cameraService.StartHeartbeatLoop(heartbeatCtx, 30*time.Second)

	// 5. 组装 HTTP 路由与服务器
	router := api.SetupRouter(db, cameraService, storageService, recordEngine, streamEngine)
	addr := fmt.Sprintf("0.0.0.0:%d", *portFlag)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("[Server] iStore NVR 管理服务已在 http://%s 启动就绪\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Fatal] 服务启动异常: %v\n", err)
		}
	}()

	// 6. 优雅关机信号监听
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[Shutdown] 收到关机信号，正在平滑停止录像切片与资源回收...")
	// 关闭所有正在执行的录像任务，确保当前切片正常收尾
	_ = recordEngine.SetGlobalRecordSwitch(false)
	streamEngine.Close()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Shutdown] 服务退出异常: %v\n", err)
	}

	log.Println("[Shutdown] iStore NVR 核心服务已安全退出")
}
