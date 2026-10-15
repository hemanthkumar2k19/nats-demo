package api

import (
	"net/http"
	"services/model"
	"services/service"

	"github.com/gin-gonic/gin"
)

// ConsumerHandler handles HTTP endpoints for JetStream consumer management using Gin.
type ConsumerHandler struct {
	svc *service.Service
}

// NewConsumerHandler constructs a new ConsumerHandler instance.
func NewConsumerHandler(svc *service.Service) *ConsumerHandler {
	return &ConsumerHandler{
		svc: svc,
	}
}

// CreateConsumerHandler handles POST /api/v1/consumers.
func (h *ConsumerHandler) CreateConsumerHandler(c *gin.Context) {
	var req model.CreateConsumerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body: " + err.Error()})
		return
	}

	if req.StreamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'streamName' is required"})
		return
	}
	if req.ConsumerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'consumerName' is required"})
		return
	}

	info, err := h.svc.CreateConsumer(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

// DeleteConsumerHandler handles DELETE /api/v1/consumers using JSON body.
func (h *ConsumerHandler) DeleteConsumerHandler(c *gin.Context) {
	var req model.CreateConsumerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body: " + err.Error()})
		return
	}

	if req.StreamName == "" || req.ConsumerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fields 'streamName' and 'consumerName' are required"})
		return
	}

	if err := h.svc.DeleteConsumer(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "deleted",
		"streamName":   req.StreamName,
		"consumerName": req.ConsumerName,
	})
}

// RegisterRoutes registers consumer API routes on the Gin router group.
func (h *ConsumerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/consumers", h.CreateConsumerHandler)
	rg.DELETE("/consumers", h.DeleteConsumerHandler)
}
