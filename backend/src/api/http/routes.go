package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers the handlers onto the HTTP router.
func RegisterRoutes(router *gin.Engine, h *Handler) {
	router.Use(CORSMiddleware())
	router.POST("/jobs", h.SubmitJob)
	router.POST("/jobs/validate", h.ValidateJob)
	router.GET("/jobs", h.ListJobs)
	router.GET("/jobs/:job_id", h.GetJob)
	router.GET("/activities", h.GetActivities)
	router.POST("/jobs/replay", h.ReplayJobs)
	router.GET("/status", h.GetStatus)
	router.GET("/messaging/subscriptions", h.GetSubscriptions)
	router.GET("/messaging/activity", h.GetAddressingActivity)
	router.PUT("/processor/state", h.PutProcessorState)
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

