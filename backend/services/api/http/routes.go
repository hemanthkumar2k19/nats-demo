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
	router.POST("/jobs/schedule", h.ScheduleJob)
	router.POST("/jobs/validate", h.ValidateJob)
	router.GET("/jobs", h.ListJobs)
	router.GET("/jobs/:job_id", h.GetJob)
	router.POST("/jobs/queue", h.SubmitQueueJobs)
	router.POST("/jobs/stream", h.SubmitStreamJobs)
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
