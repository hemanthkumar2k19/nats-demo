package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterControlRoutes registers demo inspection and UI control endpoints onto the router.
func RegisterControlRoutes(router *gin.Engine, h *ControlHandler) {
	router.Use(CORSMiddleware())
	router.GET("/status", h.GetStatus)
	router.GET("/activities", h.GetActivities)
	router.DELETE("/activities", h.ClearActivities)
	router.POST("/jobs/replay", h.ReplayJobs)
	router.GET("/messaging/subscriptions", h.GetSubscriptions)
	router.GET("/messaging/activity", h.GetAddressingActivity)
	router.PUT("/processor/state", h.PutProcessorState)
	router.GET("/consumer", h.GetConsumerStatus)
	router.GET("/dlq/status", h.GetDLQStatus)
	router.POST("/dlq/setup", h.SetupDLQ)
	router.POST("/dlq/cleanup", h.CleanupDLQ)
	router.GET("/dlq/messages", h.GetDLQMessages)
	router.POST("/dlq/reprocess", h.ReprocessDLQ)
	router.POST("/dlq/purge", h.PurgeDLQ)
	router.GET("/queue-group", h.GetQueueGroupStatus)
	router.PUT("/queue-group", h.PutQueueGroupConfig)
	router.POST("/queue-group/reset", h.PostQueueGroupReset)
	router.POST("/jobs/queue", h.PublishQueueJobs)
	router.POST("/jobs/stream", h.PublishStreamJobs)
}

// CORSMiddleware returns a Gin HandlerFunc that configures CORS headers.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Correlation-Id")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
