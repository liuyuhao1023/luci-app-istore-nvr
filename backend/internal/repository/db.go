package repository

import (
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"istore-nvr/internal/model"
	"istore-nvr/internal/utils"
)

var DB *gorm.DB

// InitDB 初始化 SQLite 数据库与数据表
func InitDB(dbPath string) (*gorm.DB, error) {
	if dbPath == "" {
		// 默认优先使用挂载数据盘，若本地开发则使用当前目录 data/nvr.db
		dbPath = "data/nvr.db"
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	// 自动迁移所有数据模型
	if err := db.AutoMigrate(
		&model.Camera{},
		&model.Storage{},
		&model.RecordingIndex{},
		&model.SystemConfig{},
		&model.AlertEvent{},
		&model.User{},
	); err != nil {
		return nil, err
	}

	DB = db
	seedDefaultData(db)

	return db, nil
}

// seedDefaultData 初始化系统默认配置和默认存储
func seedDefaultData(db *gorm.DB) {
	// 1. 初始化系统全局配置
	var confCount int64
	db.Model(&model.SystemConfig{}).Count(&confCount)
	if confCount == 0 {
		defaultConf := model.SystemConfig{
			GlobalRecordEnabled: true,
			ServerPort:          8080,
			RTSPListenPort:      8554,
			WebRTCPort:          8889,
			AutoCleanCron:       "0 3 * * *",
		}
		db.Create(&defaultConf)
		log.Println("[DB] 初始化默认系统全局配置完成")
	}

	// 2. 初始化默认存储
	var storageCount int64
	db.Model(&model.Storage{}).Count(&storageCount)
	if storageCount == 0 {
		mountPoint := "./recordings"

		defaultStorage := model.Storage{
			Name:                  "本地主存储",
			Type:                  model.StorageTypeLocal,
			MountPoint:            mountPoint,
			Status:                model.StorageStatusMounted,
			AlertThresholdPercent: 90,
			AutoCleanEnabled:      true,
			MinRetentionDays:      3,
		}
		db.Create(&defaultStorage)
		log.Println("[DB] 初始化默认本地存储位置完成:", mountPoint)
	}

	// 3. 初始化默认管理员账号 admin / admin123
	var userCount int64
	db.Model(&model.User{}).Count(&userCount)
	if userCount == 0 {
		hashedPass, _ := utils.EncryptPassword("admin123")
		adminUser := model.User{
			Username: "admin",
			Password: hashedPass,
			Role:     model.RoleAdmin,
		}
		db.Create(&adminUser)
		log.Println("[DB] 初始化默认管理员账号完成: admin")
	}
}
