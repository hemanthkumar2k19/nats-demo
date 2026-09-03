package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterJobRoutes registers pure business job endpoints onto the router.
func RegisterJobRoutes(router *gin.Engine, h *JobHandler) {
	router.Use(CORSMiddleware())
	router.GET("/health", h.HealthCheck)
	router.POST("/jobs", h.SubmitJob)
	router.POST("/jobs/validate", h.ValidateJob)
	router.GET("/jobs", h.ListJobs)
	router.GET("/jobs/:job_id", h.GetJob)
	router.POST("/jobs/queue", h.SubmitQueueJobs)
	router.POST("/jobs/stream", h.SubmitStreamJobs)
}

// RegisterControlRoutes registers demo inspection and UI control endpoints onto the router.
func RegisterControlRoutes(router *gin.Engine, h *ControlHandler) {
	router.Use(CORSMiddleware())
	router.GET("/status", h.GetStatus)
	router.GET("/activities", h.GetActivities)
	router.POST("/jobs/replay", h.ReplayJobs)
	router.GET("/messaging/subscriptions", h.GetSubscriptions)
	router.GET("/messaging/activity", h.GetAddressingActivity)
	router.PUT("/processor/state", h.PutProcessorState)
	router.GET("/consumer", h.GetConsumerStatus)
	router.PUT("/consumer", h.PutConsumerConfig)
	router.POST("/consumer/reset", h.PostConsumerReset)
	router.GET("/dlq/status", h.GetDLQStatus)
	router.GET("/dlq/messages", h.GetDLQMessages)
	router.POST("/dlq/reprocess", h.ReprocessDLQ)
	router.POST("/dlq/purge", h.PurgeDLQ)
	router.GET("/queue-group", h.GetQueueGroupStatus)
	router.PUT("/queue-group", h.PutQueueGroupConfig)
	router.POST("/queue-group/reset", h.PostQueueGroupReset)
	router.POST("/jobs/queue", h.PublishQueueJobs)
	router.POST("/jobs/stream", h.PublishStreamJobs)
}

// RegisterRoutes registers the handlers onto the HTTP router (legacy all-in-one).
func RegisterRoutes(router *gin.Engine, h *Handler) {
	if h.JobHandler != nil {
		RegisterJobRoutes(router, h.JobHandler)
	}
	if h.ControlHandler != nil {
		RegisterControlRoutes(router, h.ControlHandler)
	}
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
