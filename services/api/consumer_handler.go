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

// ReplayMessagesHandler handles POST /api/v1/consumers/replay.
// It supports sequence-based replay (when startSequence is set) and timestamp-based replay (when startTime is set).
func (h *ConsumerHandler) ReplayMessagesHandler(c *gin.Context) {
	var req model.ReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body: " + err.Error()})
		return
	}

	if req.StreamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'streamName' is required"})
		return
	}

	hasSeq := req.StartSequence > 0 || req.EndSequence > 0
	hasTime := req.StartTime != nil && !req.StartTime.IsZero()

	if !hasSeq && !hasTime {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either sequence range ('startSequence' and 'endSequence') or timestamp range ('startTime') must be provided"})
		return
	}

	if hasSeq && hasTime {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot specify both sequence range and timestamp range; provide either 'startSequence'/'endSequence' or 'startTime'/'endTime'"})
		return
	}

	if hasSeq {
		if req.StartSequence == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'startSequence' must be greater than 0"})
			return
		}
		if req.EndSequence < req.StartSequence {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'endSequence' must be greater than or equal to 'startSequence'"})
			return
		}
	}

	if hasTime {
		if req.EndTime != nil && !req.EndTime.IsZero() && req.EndTime.Before(*req.StartTime) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'endTime' cannot be earlier than 'startTime'"})
			return
		}
	}

	resp, err := h.svc.ReplayMessages(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ReprocessMessagesHandler handles POST /api/v1/consumers/reprocess.
// It supports sequence-based reprocessing and timestamp-based reprocessing, executing domain business effects for each message.
func (h *ConsumerHandler) ReprocessMessagesHandler(c *gin.Context) {
	var req model.ReprocessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body: " + err.Error()})
		return
	}

	if req.StreamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'streamName' is required"})
		return
	}

	hasSeq := req.StartSequence > 0 || req.EndSequence > 0
	hasTime := req.StartTime != nil && !req.StartTime.IsZero()

	if !hasSeq && !hasTime {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either sequence range ('startSequence' and 'endSequence') or timestamp range ('startTime') must be provided"})
		return
	}

	if hasSeq && hasTime {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot specify both sequence range and timestamp range; provide either 'startSequence'/'endSequence' or 'startTime'/'endTime'"})
		return
	}

	if hasSeq {
		if req.StartSequence == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'startSequence' must be greater than 0"})
			return
		}
		if req.EndSequence < req.StartSequence {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'endSequence' must be greater than or equal to 'startSequence'"})
			return
		}
	}

	if hasTime {
		if req.EndTime != nil && !req.EndTime.IsZero() && req.EndTime.Before(*req.StartTime) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'endTime' cannot be earlier than 'startTime'"})
			return
		}
	}

	resp, err := h.svc.ReprocessMessages(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RegisterRoutes registers consumer API routes on the Gin router group.
func (h *ConsumerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/consumers", h.CreateConsumerHandler)
	rg.DELETE("/consumers", h.DeleteConsumerHandler)
	rg.POST("/consumers/replay", h.ReplayMessagesHandler)
	rg.POST("/consumers/reprocess", h.ReprocessMessagesHandler)
}
