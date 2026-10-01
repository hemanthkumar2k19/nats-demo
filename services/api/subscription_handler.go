package api

import (
	"net/http"
	"services/model"
	"services/service"

	"github.com/gin-gonic/gin"
)

// SubscriptionHandler handles HTTP endpoints for managing NATS subscription lifecycles using Gin.
type SubscriptionHandler struct {
	service *service.Service
}

// NewSubscriptionHandler constructs a new SubscriptionHandler instance.
func NewSubscriptionHandler(svc *service.Service) *SubscriptionHandler {
	return &SubscriptionHandler{
		service: svc,
	}
}

// SubscribeHandler handles POST /api/v1/subscriptions (create subscription).
func (h *SubscriptionHandler) SubscribeHandler(c *gin.Context) {
	var req model.SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}
	if req.Subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'subject' is required"})
		return
	}

	item, err := h.service.CreateSubscription(req.Subject, req.HandlerType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// ListSubscriptionsHandler handles GET /api/v1/subscriptions (list all subscriptions).
func (h *SubscriptionHandler) ListSubscriptionsHandler(c *gin.Context) {
	items := h.service.ListSubscriptions()
	c.JSON(http.StatusOK, items)
}

// UnsubscribeHandler handles POST /api/v1/subscriptions/unsubscribe.
func (h *SubscriptionHandler) UnsubscribeHandler(c *gin.Context) {
	var req model.SubscriptionActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}
	if req.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'id' is required"})
		return
	}

	if err := h.service.Unsubscribe(req.ID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "unsubscribed", "id": req.ID})
}

// DrainSubscriptionHandler handles POST /api/v1/subscriptions/drain.
func (h *SubscriptionHandler) DrainSubscriptionHandler(c *gin.Context) {
	var req model.SubscriptionActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON body"})
		return
	}
	if req.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'id' is required"})
		return
	}

	if err := h.service.DrainSubscription(req.ID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "draining", "id": req.ID})
}

// RegisterRoutes registers subscription API routes on the Gin router group.
func (h *SubscriptionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/subscriptions", h.SubscribeHandler)
	rg.GET("/subscriptions", h.ListSubscriptionsHandler)
	rg.POST("/subscriptions/unsubscribe", h.UnsubscribeHandler)
	rg.POST("/subscriptions/drain", h.DrainSubscriptionHandler)
}
