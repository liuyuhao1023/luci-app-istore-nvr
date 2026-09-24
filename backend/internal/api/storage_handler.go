package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"istore-nvr/internal/model"
	"istore-nvr/internal/service"
)

type StorageHandler struct {
	storageService *service.StorageService
}

func NewStorageHandler(ss *service.StorageService) *StorageHandler {
	return &StorageHandler{storageService: ss}
}

// ListStorages 获取所有存储设备 (包含本地盘与 SMB 网络存储)
func (h *StorageHandler) ListStorages(c *gin.Context) {
	list, err := h.storageService.ListStorages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

type AddStorageReq struct {
	model.Storage
	Password string `json:"password"`
}

// AddStorage 添加存储设备 (本地或 SMB)
func (h *StorageHandler) AddStorage(c *gin.Context) {
	var req AddStorageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	st, err := h.storageService.AddStorage(&req.Storage, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "添加成功", "data": st})
}

// UpdateStorage 更新存储配置
func (h *StorageHandler) UpdateStorage(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req AddStorageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	st, err := h.storageService.UpdateStorage(uint(id), &req.Storage, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": st})
}

// DeleteStorage 删除存储设备
func (h *StorageHandler) DeleteStorage(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.storageService.DeleteStorage(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// MountSMB 手动测试/重新挂载 SMB 存储
func (h *StorageHandler) MountSMB(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	st, plainPass, err := h.storageService.GetStorageByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "存储设备不存在"})
		return
	}

	if st.Type != model.StorageTypeSMB {
		c.JSON(http.StatusBadRequest, gin.H{"error": "非 SMB 网络存储，无需挂载"})
		return
	}

	if err := h.storageService.MountSMB(st, plainPass); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "SMB 网络共享目录挂载与读写测试成功", "data": st})
}
