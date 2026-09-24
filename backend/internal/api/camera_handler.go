package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"istore-nvr/internal/model"
	"istore-nvr/internal/service"
)

type CameraHandler struct {
	camService    *service.CameraService
	recordEngine  *service.RecordEngine
}

func NewCameraHandler(cs *service.CameraService, re *service.RecordEngine) *CameraHandler {
	return &CameraHandler{
		camService:   cs,
		recordEngine: re,
	}
}

// ListCameras 获取摄像头列表
func (h *CameraHandler) ListCameras(c *gin.Context) {
	group := c.Query("group")
	keyword := c.Query("keyword")

	var onlineFilter *bool
	if onlineStr := c.Query("online"); onlineStr != "" {
		isOnline := onlineStr == "true" || onlineStr == "1"
		onlineFilter = &isOnline
	}

	list, err := h.camService.ListCameras(group, onlineFilter, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 动态注入真实录像任务状态 (针对 REC 标识)
	for i := range list {
		list[i].IsRecording = h.recordEngine.IsCameraRecording(list[i].ID)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// GetCamera 获取单台摄像头信息
func (h *CameraHandler) GetCamera(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	cam, _, err := h.camService.GetCameraByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "摄像头不存在"})
		return
	}
	cam.IsRecording = h.recordEngine.IsCameraRecording(cam.ID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cam})
}

type AddCameraReq struct {
	model.Camera
	Password string `json:"password"`
}

// AddCamera 手动添加摄像头
func (h *CameraHandler) AddCamera(c *gin.Context) {
	var req AddCameraReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cam, err := h.camService.AddCamera(&req.Camera, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "添加成功", "data": cam})
}

// UpdateCamera 更新摄像头
func (h *CameraHandler) UpdateCamera(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req AddCameraReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cam, err := h.camService.UpdateCamera(uint(id), &req.Camera, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": cam})
}

// DeleteCamera 删除摄像头
func (h *CameraHandler) DeleteCamera(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	_ = h.recordEngine.StopCameraRecord(uint(id))
	if err := h.camService.DeleteCamera(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestConnection 跨网段快速测试摄像头
func (h *CameraHandler) TestConnection(c *gin.Context) {
	var req AddCameraReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	info, err := h.camService.TestConnection(&req.Camera, req.Password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "message": err.Error(), "data": info})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "连接测试成功", "data": info})
}

type ScanReq struct {
	Subnets      []string `json:"subnets"`       // 用户输入的跨网段列表，如 ["192.168.1.0/24", "192.168.2.0/24"]
	UseMulticast bool     `json:"use_multicast"` // 是否同时启用本地组播 WS-Discovery
}

// ScanDevices 触发跨网段扫描与自动探测
func (h *CameraHandler) ScanDevices(c *gin.Context) {
	var req ScanReq
	_ = c.ShouldBindJSON(&req)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	devices, err := h.camService.ScanDevices(ctx, req.Subnets, req.UseMulticast)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": devices, "count": len(devices)})
}

// ToggleCameraRecord 独立控制单台摄像头录像开关 (第7.2节需求)
func (h *CameraHandler) ToggleCameraRecord(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var err error
	if req.Enabled {
		err = h.recordEngine.StartCameraRecord(uint(id))
	} else {
		err = h.recordEngine.StopCameraRecord(uint(id))
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "录像开关已更新",
		"enabled": req.Enabled,
	})
}

// BatchRecord 批量操作录像开关
func (h *CameraHandler) BatchRecord(c *gin.Context) {
	var req struct {
		IDs     []uint `json:"ids"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.camService.BatchUpdateRecord(req.IDs, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 触发录像引擎任务变更
	for _, id := range req.IDs {
		if req.Enabled {
			_ = h.recordEngine.StartCameraRecord(id)
		} else {
			_ = h.recordEngine.StopCameraRecord(id)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "批量更新成功"})
}

// BatchStorage 批量变更摄像头录像存储位置 (第17.4节需求)
func (h *CameraHandler) BatchStorage(c *gin.Context) {
	var req struct {
		IDs       []uint `json:"ids"`
		StorageID uint   `json:"storage_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.camService.BatchUpdateStorage(req.IDs, req.StorageID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "批量变更存储位置成功"})
}
