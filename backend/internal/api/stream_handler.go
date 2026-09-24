package api

import (
	"net"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"istore-nvr/internal/service"
)

type StreamHandler struct {
	streamEngine *service.StreamEngine
}

func NewStreamHandler(se *service.StreamEngine) *StreamHandler {
	return &StreamHandler{streamEngine: se}
}

// RequestStream 请求单路摄像头实时预览流信息 (返回 WebRTC WHEP / HLS 地址)
func (h *StreamHandler) RequestStream(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	streamType := c.DefaultQuery("stream", "sub") // 默认请求子码流，保护性能

	// 获取客户端访问软路由时使用的 Host/IP (如 192.168.1.15)
	host := c.Request.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	streamInfo, err := h.streamEngine.RequestStream(uint(id), streamType, host)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": streamInfo,
	})
}

// ReleaseStream 释放预览流引用
func (h *StreamHandler) ReleaseStream(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	streamType := c.DefaultQuery("stream", "sub")
	h.streamEngine.ReleaseStream(uint(id), streamType)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已释放视频流引用"})
}
