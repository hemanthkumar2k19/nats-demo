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

// PublishHandler handles HTTP POST /api/v1/publish requests.
func (h *PublisherHandler) PublishHandler(c *gin.Context) {
	var msg model.Message
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	if msg.Subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'subject' is required"})
		return
	}

	var err error
	if msg.Reply != "" {
		err = h.pubService.PublishWithReply(&msg)
	} else {
		err = h.pubService.Publish(&msg)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.PublishResponse{
		Status:  "published",
		Subject: msg.Subject,
	})
}

// PublishRequestHandler handles HTTP POST /api/v1/publish/request requests.
func (h *PublisherHandler) PublishRequestHandler(c *gin.Context) {
	var msg model.Message
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	if msg.Subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'subject' is required"})
		return
	}

	resp, err := h.pubService.PublishRequest(&msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RegisterRoutes registers publisher API routes on the Gin router group.
func (h *PublisherHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/publish", h.PublishHandler)
	rg.POST("/publish/request", h.PublishRequestHandler)
}
