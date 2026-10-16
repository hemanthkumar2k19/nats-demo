package api

import (
	"net/http"
	"services/model"
	"services/service"

	"github.com/gin-gonic/gin"
)

// StreamHandler handles HTTP endpoints for JetStream stream management using Gin.
type StreamHandler struct {
	svc *service.Service
}

// NewStreamHandler constructs a new StreamHandler instance.
func NewStreamHandler(svc *service.Service) *StreamHandler {
	return &StreamHandler{
		svc: svc,
	}
}

// CreateStreamHandler handles POST /api/v1/streams.
func (h *StreamHandler) CreateStreamHandler(c *gin.Context) {
	var req model.CreateStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body: " + err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'name' is required"})
		return
	}

	info, err := h.svc.CreateStream(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

// DeleteStreamHandler handles DELETE /api/v1/streams using JSON body.
func (h *StreamHandler) DeleteStreamHandler(c *gin.Context) {
	var req model.DeleteStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body: " + err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'name' is required"})
		return
	}

	if err := h.svc.DeleteStream(c.Request.Context(), req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "deleted",
		"name":   req.Name,
	})
}

// RegisterRoutes registers stream API routes on the Gin router group.
func (h *StreamHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/streams", h.CreateStreamHandler)
	rg.DELETE("/streams", h.DeleteStreamHandler)
}
