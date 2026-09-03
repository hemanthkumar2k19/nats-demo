package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"nats-demo/internal/jobs"
	"nats-demo/internal/messaging"
	"nats-demo/internal/telemetry"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// JobServiceDomain defines domain operations required by JobHandler.
type JobServiceDomain interface {
	SubmitJob(ctx context.Context, job jobs.Job) (*jobs.JobStatusResponse, error)
	ValidateJob(ctx context.Context, job jobs.Job) (*jobs.JobValidationResponse, error)
	ListJobs() []*jobs.JobDetailResponse
	GetJob(jobID string) (*jobs.JobDetailResponse, bool)
	SubmitQueueJobs(ctx context.Context, req jobs.QueuePublishRequest) (*jobs.QueuePublishResponse, error)
	SubmitStreamJobs(ctx context.Context, req jobs.QueuePublishRequest) (*jobs.QueuePublishResponse, error)
}

// JobHandler handles HTTP requests for the pure business Job Service.
type JobHandler struct {
	jobService JobServiceDomain
}

// NewJobHandler instantiates a new JobHandler.
func NewJobHandler(jobService JobServiceDomain) *JobHandler {
	return &JobHandler{
		jobService: jobService,
	}
}

// HealthCheck returns health status for job-service.
func (h *JobHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "UP",
		"service": "job-service",
	})
}

// SubmitJob handles job submission HTTP POST requests.
func (h *JobHandler) SubmitJob(c *gin.Context) {
	start := time.Now()
	spanCtx, span := telemetry.StartSpan(c.Request.Context(), "POST /jobs",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("http.method", "POST"),
			attribute.String("http.route", "/jobs"),
		),
	)
	defer span.End()

	var job jobs.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	span.SetAttributes(
		attribute.String("job.id", job.JobID),
		attribute.String("job.type", job.Type),
	)

	if job.JobID == "" {
		span.SetStatus(codes.Error, "missing job_id")
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	if span.SpanContext().HasTraceID() {
		job.TraceID = span.SpanContext().TraceID().String()
	}

	resp, err := h.jobService.SubmitJob(spanCtx, job)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		telemetry.RecordJobSubmission(spanCtx, job.DeliveryMode, "FAILED", time.Since(start))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit job: " + err.Error()})
		return
	}

	telemetry.RecordJobSubmission(spanCtx, job.DeliveryMode, "SUCCESS", time.Since(start))
	c.JSON(http.StatusAccepted, resp)
}

// ValidateJob handles synchronous job validation requests using NATS Request/Reply.
func (h *JobHandler) ValidateJob(c *gin.Context) {
	start := time.Now()
	spanCtx, span := telemetry.StartSpan(c.Request.Context(), "POST /jobs/validate",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("http.method", "POST"),
			attribute.String("http.route", "/jobs/validate"),
		),
	)
	defer span.End()

	var job jobs.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	span.SetAttributes(
		attribute.String("job.id", job.JobID),
		attribute.String("job.type", job.Type),
	)

	if job.JobID == "" {
		span.SetStatus(codes.Error, "missing job_id")
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}
	if job.Type == "" {
		span.SetStatus(codes.Error, "missing type")
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
		return
	}

	if span.SpanContext().HasTraceID() {
		job.TraceID = span.SpanContext().TraceID().String()
	}

	resp, err := h.jobService.ValidateJob(spanCtx, job)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		telemetry.RecordValidationRequest(spanCtx, "ERROR", time.Since(start))

		if errors.Is(err, messaging.ErrRequestTimeout) {
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error": "validation request timed out: no processor replied within 2s",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate job: " + err.Error(),
		})
		return
	}

	resultStr := "INVALID"
	if resp.Valid {
		resultStr = "VALID"
	}
	telemetry.RecordValidationRequest(spanCtx, resultStr, time.Since(start))

	c.JSON(http.StatusOK, resp)
}

// ListJobs returns the list of all tracked jobs.
func (h *JobHandler) ListJobs(c *gin.Context) {
	jobsList := h.jobService.ListJobs()
	c.JSON(http.StatusOK, gin.H{
		"jobs": jobsList,
	})
}

// GetJob returns detailed status and history of a single job.
func (h *JobHandler) GetJob(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	job, exists := h.jobService.GetJob(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// SubmitQueueJobs handles batch or single job submission to Core NATS jobs.queue subject.
func (h *JobHandler) SubmitQueueJobs(c *gin.Context) {
	start := time.Now()
	spanCtx, span := telemetry.StartSpan(c.Request.Context(), "POST /jobs/queue",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("http.method", "POST"),
			attribute.String("http.route", "/jobs/queue"),
		),
	)
	defer span.End()

	var req jobs.QueuePublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Count = 10
	}
	if req.Count <= 0 {
		req.Count = 10
	}

	span.SetAttributes(attribute.Int("job.count", req.Count))

	resp, err := h.jobService.SubmitQueueJobs(spanCtx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		telemetry.RecordJobSubmission(spanCtx, "CORE", "FAILED", time.Since(start))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit queue jobs: " + err.Error()})
		return
	}

	telemetry.RecordJobSubmission(spanCtx, "CORE", "SUCCESS", time.Since(start))
	c.JSON(http.StatusAccepted, resp)
}

// SubmitStreamJobs publishes a batch of test jobs directly to JetStream stream JOBS.
func (h *JobHandler) SubmitStreamJobs(c *gin.Context) {
	start := time.Now()
	spanCtx, span := telemetry.StartSpan(c.Request.Context(), "POST /jobs/stream",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("http.method", "POST"),
			attribute.String("http.route", "/jobs/stream"),
		),
	)
	defer span.End()

	var req jobs.QueuePublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Count = 10
	}
	if req.Count <= 0 {
		req.Count = 10
	}

	span.SetAttributes(attribute.Int("job.count", req.Count))

	resp, err := h.jobService.SubmitStreamJobs(spanCtx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		telemetry.RecordJobSubmission(spanCtx, "JETSTREAM", "FAILED", time.Since(start))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit stream jobs: " + err.Error()})
		return
	}

	telemetry.RecordJobSubmission(spanCtx, "JETSTREAM", "SUCCESS", time.Since(start))
	c.JSON(http.StatusAccepted, resp)
}

