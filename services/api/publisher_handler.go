package api

import (
	"net/http"
	"services/model"
	"services/service"

	"github.com/gin-gonic/gin"
)

// PublisherHandler handles HTTP endpoints for publishing messages to NATS using Gin.
type PublisherHandler struct {
	pubService *service.Service
}

// NewPublisherHandler constructs a new PublisherHandler instance.
func NewPublisherHandler(pubService *service.Service) *PublisherHandler {
	return &PublisherHandler{
		pubService: pubService,
	}
}

// PublishHandler handles HTTP POST /api/v1/publish and /api/v1/publish/js.
func (h *PublisherHandler) PublishHandler(c *gin.Context) {
	var msg model.Message
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body: " + err.Error()})
		return
	}

	if msg.Subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'subject' is required"})
		return
	}

	ack, err := h.pubService.JSPublish(c.Request.Context(), &msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ack)
}

// RegisterRoutes registers publisher API routes on the Gin router group.
func (h *PublisherHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/publish/js", h.PublishHandler)
}
