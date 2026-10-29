package api

import (
	"encoding/json"
	"fmt"
	"io"
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

// PublishHandler handles HTTP POST /api/v1/publish for Core NATS.
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

	if httpMsgID := c.GetHeader("Nats-Msg-Id"); httpMsgID != "" {
		if msg.Headers == nil {
			msg.Headers = make(map[string]any)
		}
		msg.Headers["Nats-Msg-Id"] = httpMsgID
	}

	resp, err := h.pubService.Publish(c.Request.Context(), &msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// JSPublishHandler handles HTTP POST /api/v1/publish/js for JetStream.
func (h *PublisherHandler) JSPublishHandler(c *gin.Context) {
	var msg model.Message
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body: " + err.Error()})
		return
	}

	if msg.Subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'subject' is required"})
		return
	}

	if httpMsgID := c.GetHeader("Nats-Msg-Id"); httpMsgID != "" {
		if msg.Headers == nil {
			msg.Headers = make(map[string]any)
		}
		msg.Headers["Nats-Msg-Id"] = httpMsgID
	}

	resp, err := h.pubService.JSPublish(c.Request.Context(), &msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// JSPublishBatchHandler handles HTTP POST /api/v1/publish/js/batch for publishing multiple messages asynchronously.
func (h *PublisherHandler) JSPublishBatchHandler(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body: " + err.Error()})
		return
	}

	var messages []model.Message

	// 1. Support dynamic batch publish by count: {"subject": "payments.created", "count": 6}
	var countReq struct {
		Subject string `json:"subject"`
		Count   int    `json:"count"`
	}
	if err := json.Unmarshal(bodyBytes, &countReq); err == nil && countReq.Subject != "" && countReq.Count > 0 {
		messages = make([]model.Message, countReq.Count)
		for i := 0; i < countReq.Count; i++ {
			messages[i] = model.Message{
				Subject: countReq.Subject,
			}
		}
	} else if err := json.Unmarshal(bodyBytes, &messages); err != nil {
		// 2. Fallback to array of messages or {"messages": [...]}
		var req struct {
			Messages []model.Message `json:"messages"`
		}
		if err := json.Unmarshal(bodyBytes, &req); err != nil || len(req.Messages) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: provide {'subject': '...', 'count': N}, an array of messages, or {'messages': [...]}"})
			return
		}
		messages = req.Messages
	}

	for i, msg := range messages {
		if msg.Subject == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Message at index %d is missing required 'subject' field", i)})
			return
		}
	}

	if err := h.pubService.JSPublishBatch(c.Request.Context(), messages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "published",
		"count":  len(messages),
	})
}

// RegisterRoutes registers publisher API routes on the Gin router group.
func (h *PublisherHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/publish", h.PublishHandler)
	rg.POST("/publish/js", h.JSPublishHandler)
	rg.POST("/publish/js/batch", h.JSPublishBatchHandler)
}
