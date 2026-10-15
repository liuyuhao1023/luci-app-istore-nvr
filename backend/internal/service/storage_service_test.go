package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"istore-nvr/internal/model"
)

func TestCheckAndCleanStorage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nvr_test_storage_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	_ = db.AutoMigrate(&model.Camera{}, &model.Storage{}, &model.RecordingIndex{}, &model.AlertEvent{})

	st := model.Storage{
		Name:                  "TestStorage",
		Type:                  model.StorageTypeLocal,
		MountPoint:            tmpDir,
		Status:                model.StorageStatusMounted,
		AlertThresholdPercent: 10, // set low threshold to trigger cleanup in test
		AutoCleanEnabled:      true,
		MinRetentionDays:      1,
	}
	if err := db.Create(&st).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	for i := 1; i <= 5; i++ {
		dateDir := filepath.Join(tmpDir, "recordings", "1", "2026", "10", "15")
		_ = os.MkdirAll(dateDir, 0755)
		fileName := fmt.Sprintf("rec_%d.mp4", i)
		filePath := filepath.Join(dateDir, fileName)
		_ = os.WriteFile(filePath, []byte("fake video content"), 0644)

		isProtected := (i == 5) // Last one is protected
		startTime := now.Add(-time.Duration(10-i) * 24 * time.Hour) // older than MinRetentionDays

		rec := model.RecordingIndex{
			CameraID:    1,
			StorageID:   st.ID,
			FilePath:    filepath.Join("recordings", "1", "2026", "10", "15", fileName),
			Format:      "mp4",
			StartTime:   startTime,
			EndTime:     startTime.Add(5 * time.Minute),
			SizeBytes:   1024,
			IsProtected: isProtected,
		}
		_ = db.Create(&rec)
	}

	svc := NewStorageService(db)

	deletedCount, deletedBytes, err := svc.CheckAndCleanStorage(&st)
	if err != nil {
		t.Fatalf("CheckAndCleanStorage failed: %v", err)
	}

	t.Logf("Deleted count: %d, deleted bytes: %d", deletedCount, deletedBytes)

	if deletedCount != 4 {
		t.Errorf("Expected 4 deleted records, got %d", deletedCount)
	}

	// Check if protected file is kept
	var protectedRec model.RecordingIndex
	if err := db.Where("storage_id = ? AND is_protected = ?", st.ID, true).First(&protectedRec).Error; err != nil {
		t.Errorf("Protected record was deleted!")
	}

	// Check AlertEvent was recorded
	var event model.AlertEvent
	if err := db.Where("source = ? AND source_id = ?", "storage", st.ID).First(&event).Error; err != nil {
		t.Errorf("AlertEvent was not recorded!")
	} else {
		t.Logf("Recorded event: %s", event.Message)
	}
}
