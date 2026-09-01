package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"nats-demo/internal/jobs"
	"nats-demo/internal/messaging"
	"nats-demo/internal/natsclient"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

// JobService specifies the domain service functionality needed by HTTP handlers.
type JobService interface {
	SubmitJob(job jobs.Job, correlationID string) (*jobs.JobStatusResponse, error)
	ValidateJob(job jobs.Job, correlationID string) (*jobs.JobValidationResponse, error)
	ListJobs() []*jobs.JobDetailResponse
	GetJob(jobID string) (*jobs.JobDetailResponse, bool)
	GetActivities() []jobs.Activity
}

// Handler handles HTTP requests for the Demo Service.
type Handler struct {
	jobService JobService
	natsClient *natsclient.Client
	observer   *messaging.Observer
}

// NewHandler instantiates a new Handler.
func NewHandler(jobService JobService, natsClient *natsclient.Client, observer *messaging.Observer) *Handler {
	return &Handler{
		jobService: jobService,
		natsClient: natsClient,
		observer:   observer,
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
	isProcessing := false
	workers := 1
	consumerName := "job-processor"
	if natsStatus == "CONNECTED" {
		// Ping the processor using Request/Reply
		reply, err := h.natsClient.Conn.Request("status.processor", nil, 250*time.Millisecond)
		if err == nil && len(reply.Data) > 0 {
			processorStatus = "ACTIVE"
			var statusResp struct {
				Status       string `json:"status"`
				Processing   bool   `json:"processing"`
				Workers      int    `json:"workers"`
				ConsumerName string `json:"consumer_name"`
			}
			if err := json.Unmarshal(reply.Data, &statusResp); err == nil {
				isProcessing = statusResp.Processing
				if statusResp.Workers > 0 {
					workers = statusResp.Workers
				}
				if statusResp.ConsumerName != "" {
					consumerName = statusResp.ConsumerName
				}
			}
		}
	}

	// Fetch JetStream JOBS stream pending message count
	var jsInfo gin.H
	if natsStatus == "CONNECTED" {
		js, err := h.natsClient.Conn.JetStream()
		if err == nil {
			// Query active consumer stats
			cinfo, err := js.ConsumerInfo("JOBS", consumerName)
			if err != nil && consumerName != "job-processor" {
				cinfo, err = js.ConsumerInfo("JOBS", "job-processor")
			}
			if err != nil {
				cinfo, err = js.ConsumerInfo("JOBS", "processor-durable")
			}
			if err == nil && cinfo != nil {
				jsInfo = gin.H{
					"stream":  "JOBS",
					"pending": cinfo.NumPending,
				}
			} else {
				// Fallback to general stream message count if consumer is not created yet
				sinfo, err := js.StreamInfo("JOBS")
				if err == nil {
					jsInfo = gin.H{
						"stream":  "JOBS",
						"pending": sinfo.State.Msgs,
					}
				}
			}
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
				"name":       "processor-service",
				"status":     processorStatus,
				"processing": isProcessing,
				"workers":    workers,
			},
		},
		"jetstream": jsInfo,
	})
}

// SubmitJob handles job submission HTTP POST requests.
func (h *Handler) SubmitJob(c *gin.Context) {
	var job jobs.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if job.DeliveryMode == "" {
		job.DeliveryMode = "CORE"
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

	// For CORE delivery mode, if processor is OFF (not active or not processing),
	// publish a SubjectJobNoConsumer lifecycle event immediately to represent transient message loss.
	if job.DeliveryMode == "CORE" {
		isProcessing := false
		reply, err := h.natsClient.Conn.Request("status.processor", nil, 100*time.Millisecond)
		if err == nil {
			var processorStatus struct {
				Status     string `json:"status"`
				Processing bool   `json:"processing"`
			}
			if err := json.Unmarshal(reply.Data, &processorStatus); err == nil {
				isProcessing = processorStatus.Processing
			}
		}

		if !isProcessing {
			pub := messaging.NewPublisher(h.natsClient)
			_ = pub.PublishJobLifecycle(
				messaging.SubjectJobNoConsumer,
				job.JobID,
				"NO CONSUMER",
				1,
				"No active consumer for Core NATS message",
				correlationID,
				"demo-service",
				job.DeliveryMode,
				0,
			)
		}
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

	correlationID := c.GetHeader("X-Correlation-Id")
	if correlationID == "" {
		correlationID = "corr-val-" + job.JobID
	}

	resp, err := h.jobService.ValidateJob(job, correlationID)
	if err != nil {
		// When the processor has no active responder the request times out.
		// Return 504 so the UI can display the timeout scenario clearly.
		if errors.Is(err, messaging.ErrRequestTimeout) {
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error":   "request timed out",
				"message": "No response received from processor service",
			})
			return
		}
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

// GetSubscriptions returns the active subscriptions for the addressing demo.
func (h *Handler) GetSubscriptions(c *gin.Context) {
	subs := []gin.H{
		{"name": "exact", "subject": "jobs.submitted"},
		{"name": "single-level", "subject": "jobs.*"},
		{"name": "multi-level", "subject": "jobs.>"},
	}
	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subs,
	})
}

// GetAddressingActivity returns observed message delivery activity for the addressing demo.
func (h *Handler) GetAddressingActivity(c *gin.Context) {
	events := h.observer.GetEvents()
	c.JSON(http.StatusOK, gin.H{
		"events": events,
	})
}

// PutProcessorState handles changing the processing state of the processor service.
func (h *Handler) PutProcessorState(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	payload, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal control request"})
		return
	}

	reply, err := h.natsClient.Conn.Request(messaging.SubjectProcessorStateSet, payload, 1*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Processor service did not respond. Is it running?"})
		return
	}

	var resp struct {
		Enabled bool   `json:"enabled"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid response from processor service"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetConsumerStatus returns the current consumer configuration and live metrics.
func (h *Handler) GetConsumerStatus(c *gin.Context) {
	consumerName := "job-processor"
	consumerType := "durable"
	workers := 1
	ordering := "normal"
	status := "ACTIVE"

	// Ping processor service via status.processor to query current configuration
	reply, err := h.natsClient.Conn.Request("status.processor", nil, 500*time.Millisecond)
	if err == nil && len(reply.Data) > 0 {
		var procStatus struct {
			Status       string `json:"status"`
			Processing   bool   `json:"processing"`
			ConsumerType string `json:"consumer_type"`
			ConsumerName string `json:"consumer_name"`
			Workers      int    `json:"workers"`
			Ordering     string `json:"ordering"`
		}
		if err := json.Unmarshal(reply.Data, &procStatus); err == nil {
			if procStatus.ConsumerName != "" {
				consumerName = procStatus.ConsumerName
			}
			if procStatus.ConsumerType != "" {
				consumerType = procStatus.ConsumerType
			}
			if procStatus.Workers > 0 {
				workers = procStatus.Workers
			}
			if procStatus.Ordering != "" {
				ordering = procStatus.Ordering
			}
			if !procStatus.Processing {
				status = "STOPPED"
			}
		}
	} else {
		status = "OFFLINE"
	}

	var pending uint64
	var ackPending int
	var redelivered int

	js, err := h.natsClient.Conn.JetStream()
	if err == nil {
		cinfo, err := js.ConsumerInfo("JOBS", consumerName)
		if err != nil && consumerName != "processor-durable" {
			cinfo, _ = js.ConsumerInfo("JOBS", "processor-durable")
		}
		if cinfo != nil {
			pending = cinfo.NumPending
			ackPending = cinfo.NumAckPending
			redelivered = cinfo.NumRedelivered
		}
	}

	c.JSON(http.StatusOK, jobs.ConsumerStatusResponse{
		Name:        consumerName,
		Type:        consumerType,
		Workers:     workers,
		Ordering:    ordering,
		Delivery:    "at-least-once",
		Status:      status,
		Pending:     pending,
		AckPending:  ackPending,
		Redelivered: redelivered,
	})
}

// PutConsumerConfig reconfigures the consumer settings (durable vs ephemeral, workers, ordering).
func (h *Handler) PutConsumerConfig(c *gin.Context) {
	var req jobs.ConsumerConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	if req.Type == "" {
		req.Type = "durable"
	}
	if req.Workers <= 0 {
		req.Workers = 1
	}
	if req.Ordering == "" {
		req.Ordering = "normal"
	}
	if req.Ordering == "ordered" {
		req.Workers = 1
	}

	payload, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal consumer config payload"})
		return
	}

	reply, err := h.natsClient.Conn.Request(messaging.SubjectConsumerConfigSet, payload, 2*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Processor service did not respond to consumer config update. Is it running?"})
		return
	}

	var resp jobs.ConsumerStatusResponse
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid response from processor service"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

