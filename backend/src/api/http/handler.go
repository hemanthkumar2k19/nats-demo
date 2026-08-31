package http

import (
	"net/http"
	"time"

	"nats-demo/internal/jobs"
	"nats-demo/internal/natsclient"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

// JobService specifies the domain service functionality needed by HTTP handlers.
type JobService interface {
	SubmitJob(job jobs.Job, correlationID string) (*jobs.JobStatusResponse, error)
	ValidateJob(job jobs.Job) (*jobs.JobValidationResponse, error)
	ListJobs() []*jobs.JobDetailResponse
	GetJob(jobID string) (*jobs.JobDetailResponse, bool)
	GetActivities() []jobs.Activity
}

// Handler handles HTTP requests for the Demo Service.
type Handler struct {
	jobService JobService
	natsClient *natsclient.Client
}

// NewHandler instantiates a new Handler.
func NewHandler(jobService JobService, natsClient *natsclient.Client) *Handler {
	return &Handler{
		jobService: jobService,
		natsClient: natsClient,
	}
}

// GetStatus checks and returns status of NATS server and backend services.
func (h *Handler) GetStatus(c *gin.Context) {
	natsStatus := "DISCONNECTED"
	if h.natsClient != nil && h.natsClient.Conn != nil {
		switch h.natsClient.Conn.Status() {
		case nats.CONNECTED:
			natsStatus = "CONNECTED"
		case nats.CONNECTING:
			natsStatus = "CONNECTING"
		case nats.RECONNECTING:
			natsStatus = "RECONNECTING"
		default:
			natsStatus = "DISCONNECTED"
		}
	}

	processorStatus := "OFFLINE"
	if natsStatus == "CONNECTED" {
		// Ping the processor using Request/Reply
		reply, err := h.natsClient.Conn.Request("status.processor", nil, 250*time.Millisecond)
		if err == nil && len(reply.Data) > 0 {
			processorStatus = "ACTIVE"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
		"nats": gin.H{
			"status": natsStatus,
		},
		"services": []gin.H{
			{
				"name":   "demo-service",
				"status": "ACTIVE",
			},
			{
				"name":   "processor-service",
				"status": processorStatus,
			},
		},
	})
}

// SubmitJob handles job submission HTTP POST requests.
func (h *Handler) SubmitJob(c *gin.Context) {
	var job jobs.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	correlationID := c.GetHeader("X-Correlation-Id")
	if correlationID == "" {
		correlationID = "corr-" + job.JobID
	}

	resp, err := h.jobService.SubmitJob(job, correlationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, resp)
}

// ValidateJob handles sync validation requests using Request/Reply.
func (h *Handler) ValidateJob(c *gin.Context) {
	var job jobs.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	resp, err := h.jobService.ValidateJob(job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListJobs returns the list of all tracked jobs.
func (h *Handler) ListJobs(c *gin.Context) {
	c.JSON(http.StatusOK, h.jobService.ListJobs())
}

// GetJob returns detailed status of a specific job.
func (h *Handler) GetJob(c *gin.Context) {
	jobID := c.Param("job_id")
	job, exists := h.jobService.GetJob(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// GetActivities returns flat activity logs for the dashboard.
func (h *Handler) GetActivities(c *gin.Context) {
	c.JSON(http.StatusOK, h.jobService.GetActivities())
}

// ReplayJobs triggers a JetStream replay (stubbed/mocked response).
func (h *Handler) ReplayJobs(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{
		"status":   "REPLAY_STARTED",
		"consumer": "job-replay-001",
	})
}

