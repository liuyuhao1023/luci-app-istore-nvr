package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"istore-nvr/internal/model"
	"istore-nvr/internal/service"
)

type RecordHandler struct {
	db           *gorm.DB
	recordEngine *service.RecordEngine
}

func NewRecordHandler(db *gorm.DB, re *service.RecordEngine) *RecordHandler {
	return &RecordHandler{db: db, recordEngine: re}
}

// GetGlobalRecordSwitch 获取全局录像开关 (第7.1节需求)
func (h *RecordHandler) GetGlobalRecordSwitch(c *gin.Context) {
	enabled := h.recordEngine.IsGlobalEnabled()
	c.JSON(http.StatusOK, gin.H{"code": 0, "enabled": enabled})
}

// SetGlobalRecordSwitch 设置全局录像开关 (第7.1节需求)
func (h *RecordHandler) SetGlobalRecordSwitch(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.recordEngine.SetGlobalRecordSwitch(req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "全局录像开关已保存", "enabled": req.Enabled})
}

// QueryRecordings 检索录像切片列表 (第8节录像检索需求)
func (h *RecordHandler) QueryRecordings(c *gin.Context) {
	camID, _ := strconv.Atoi(c.Query("camera_id"))
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	var start, end time.Time
	if startTimeStr != "" {
		start, _ = time.Parse(time.RFC3339, startTimeStr)
	}
	if endTimeStr != "" {
		end, _ = time.Parse(time.RFC3339, endTimeStr)
	}

	records, err := h.recordEngine.QueryRecordings(uint(camID), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records, "count": len(records)})
}

// DownloadRecording 下载指定录像切片文件
func (h *RecordHandler) DownloadRecording(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var rec model.RecordingIndex
	if err := h.db.First(&rec, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "录像记录不存在"})
		return
	}

	var storage model.Storage
	if err := h.db.First(&storage, rec.StorageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "存储设备不存在"})
		return
	}

	absPath := filepath.Join(storage.MountPoint, rec.FilePath)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "磁盘上的录像文件已不存在或已被清理"})
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(absPath)))
	c.File(absPath)
}
