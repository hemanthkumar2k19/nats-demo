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

// JSPublishHandler handles HTTP POST /api/v1/publish/js or /api/v1/js/publish (JetStream Publish).
func (h *PublisherHandler) JSPublishHandler(c *gin.Context) {
	var msg model.Message
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
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

// JSPublishAsyncHandler handles HTTP POST /api/v1/js/publish/async (JetStream Async Publish).
func (h *PublisherHandler) JSPublishAsyncHandler(c *gin.Context) {
	var msg model.Message
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}

	if msg.Subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'subject' is required"})
		return
	}

	ack, err := h.pubService.JSPublishAsync(c.Request.Context(), &msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ack)
}

// JSPublishPendingHandler handles HTTP GET /api/v1/js/publish/pending.
func (h *PublisherHandler) JSPublishPendingHandler(c *gin.Context) {
	pending := h.pubService.JSPublishAsyncPending()
	c.JSON(http.StatusOK, pending)
}

// RegisterRoutes registers publisher API routes on the Gin router group.
func (h *PublisherHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/publish", h.PublishHandler)
	rg.POST("/publish/request", h.PublishRequestHandler)

	// JetStream Publish Endpoints
	rg.POST("/publish/js", h.JSPublishHandler)
	rg.POST("/js/publish", h.JSPublishHandler)
	rg.POST("/js/publish/async", h.JSPublishAsyncHandler)
	rg.GET("/js/publish/pending", h.JSPublishPendingHandler)
}

