package api

import (
	"errors"
	"net/http"
	"services/model"
	"services/service"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go/jetstream"
)

// StreamHandler handles HTTP endpoints for stream state inspection using Gin.
type StreamHandler struct {
	svc *service.Service
}

// NewStreamHandler constructs a new StreamHandler instance.
func NewStreamHandler(svc *service.Service) *StreamHandler {
	return &StreamHandler{
		svc: svc,
	}
}

// GetStreamSequencesHandler handles GET /api/v1/streams/:stream/sequences and POST /api/v1/streams/sequences.
// It returns available sequence bounds, timestamps, and message counts to guide replay requests.
func (h *StreamHandler) GetStreamSequencesHandler(c *gin.Context) {
	streamName := c.Param("stream")
	if streamName == "" {
		streamName = c.Query("stream")
	}
	if streamName == "" {
		streamName = c.Query("name")
	}
	if streamName == "" && c.Request.ContentLength > 0 {
		var req model.StreamSequencesRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			if req.StreamName != "" {
				streamName = req.StreamName
			} else if req.Name != "" {
				streamName = req.Name
			}
		}
	}

	if streamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stream name is required (path param /streams/:stream/sequences, query ?stream=, or JSON body)"})
		return
	}

	bounds, err := h.svc.GetStreamSequenceBounds(c.Request.Context(), streamName)
	if err != nil {
		if errors.Is(err, jetstream.ErrStreamNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Stream not found: " + streamName})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bounds)
}

// RegisterRoutes registers stream sequence inspection API routes on the Gin router group.
func (h *StreamHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/streams/:stream/sequences", h.GetStreamSequencesHandler)
	rg.POST("/streams/sequences", h.GetStreamSequencesHandler)
}
