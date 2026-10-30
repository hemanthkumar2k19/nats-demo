package api

import (
	"errors"
	"net/http"
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

// GetStreamSequencesHandler handles GET /api/v1/streams/:stream/sequences.
// It returns available sequence bounds, timestamps, and message counts to guide replay requests.
func (h *StreamHandler) GetStreamSequencesHandler(c *gin.Context) {
	streamName := c.Param("stream")
	if streamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stream name is required in path: /api/v1/streams/:stream/sequences"})
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
}
