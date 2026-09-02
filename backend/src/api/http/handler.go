package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"nats-demo/internal/jobs"
	"nats-demo/internal/messaging"
	"nats-demo/internal/natsclient"
	"nats-demo/internal/telemetry"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// JobService specifies the domain service functionality needed by HTTP handlers.
type JobService interface {
	SubmitJob(ctx context.Context, job jobs.Job, correlationID string) (*jobs.JobStatusResponse, error)
	ValidateJob(ctx context.Context, job jobs.Job, correlationID string) (*jobs.JobValidationResponse, error)
	ListJobs() []*jobs.JobDetailResponse
	GetJob(jobID string) (*jobs.JobDetailResponse, bool)
	GetActivities() []jobs.Activity
}

// Handler handles HTTP requests for the Job Service.
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

	// Fetch JetStream JOBS stream metrics and consumer pending count
	var jsInfo gin.H
	if natsStatus == "CONNECTED" {
		js, err := h.natsClient.Conn.JetStream()
		if err == nil {
			var totalMsgs uint64
			var totalBytes uint64
			var firstSeq, lastSeq uint64

			sinfo, err := js.StreamInfo("JOBS")
			if err == nil && sinfo != nil {
				totalMsgs = sinfo.State.Msgs
				totalBytes = sinfo.State.Bytes
				firstSeq = sinfo.State.FirstSeq
				lastSeq = sinfo.State.LastSeq
			}

			// Query active consumer stats
			var pending uint64 = totalMsgs
			cinfo, err := js.ConsumerInfo("JOBS", consumerName)
			if err != nil && consumerName != "job-processor" {
				cinfo, err = js.ConsumerInfo("JOBS", "job-processor")
			}
			if err != nil {
				cinfo, err = js.ConsumerInfo("JOBS", "processor-durable")
			}
			if err == nil && cinfo != nil {
				pending = cinfo.NumPending
			}

			jsInfo = gin.H{
				"stream":    "JOBS",
				"messages":  totalMsgs,
				"bytes":     totalBytes,
				"first_seq": firstSeq,
				"last_seq":  lastSeq,
				"pending":   pending,
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
				"name":   "job-service",
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
	startTime := time.Now()

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

	// Start HTTP Server Span
	ctx, span := telemetry.StartSpan(c.Request.Context(), "POST /jobs",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("http.request.method", "POST"),
			attribute.String("http.route", "/jobs"),
			attribute.String("delivery.mode", job.DeliveryMode),
			attribute.String("job.id", job.JobID),
			attribute.String("job.type", job.Type),
		),
	)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	job.TraceID = traceID

	resp, err := h.jobService.SubmitJob(ctx, job, correlationID)
	duration := time.Since(startTime)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.Int("http.response.status_code", http.StatusInternalServerError))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "trace_id": traceID})
		return
	}

	span.SetStatus(codes.Ok, "submitted")
	span.SetAttributes(attribute.Int("http.response.status_code", http.StatusAccepted))

	// Record application metrics via central telemetry package
	telemetry.RecordJobSubmitted(ctx, job.DeliveryMode, job.Type, duration)
	telemetry.RecordNatsPublish(ctx, job.DeliveryMode, messaging.SubjectJobSubmitted)

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
				"job-service",
				job.DeliveryMode,
				0,
			)
		}
	}

	if resp != nil {
		resp.TraceID = traceID
	}
	c.JSON(http.StatusAccepted, resp)
}

// ValidateJob handles sync validation requests using Request/Reply.
func (h *Handler) ValidateJob(c *gin.Context) {
	startTime := time.Now()

	var job jobs.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	correlationID := c.GetHeader("X-Correlation-Id")
	if correlationID == "" {
		correlationID = "corr-val-" + job.JobID
	}

	// Start HTTP Server Span
	ctx, span := telemetry.StartSpan(c.Request.Context(), "POST /jobs/validate",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("http.request.method", "POST"),
			attribute.String("http.route", "/jobs/validate"),
			attribute.String("job.id", job.JobID),
			attribute.String("job.type", job.Type),
		),
	)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	job.TraceID = traceID

	resp, err := h.jobService.ValidateJob(ctx, job, correlationID)
	duration := time.Since(startTime)

	// Determine validation result for metric label
	result := "valid"
	if err != nil {
		result = "error"
		if errors.Is(err, messaging.ErrRequestTimeout) {
			result = "timeout"
		}
	} else if resp != nil && !resp.Valid {
		result = "invalid"
	}
	telemetry.RecordValidationRequest(ctx, result, duration)
	telemetry.RecordNatsRequest(ctx, messaging.SubjectJobValidate, duration, err)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// When the processor has no active responder the request times out.
		// Return 504 so the UI can display the timeout scenario clearly.
		if errors.Is(err, messaging.ErrRequestTimeout) {
			span.SetAttributes(attribute.Int("http.response.status_code", http.StatusGatewayTimeout))
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error":    "request timed out",
				"message":  "No response received from processor service",
				"trace_id": traceID,
			})
			return
		}
		span.SetAttributes(attribute.Int("http.response.status_code", http.StatusInternalServerError))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "trace_id": traceID})
		return
	}

	span.SetStatus(codes.Ok, "validated")
	span.SetAttributes(attribute.Int("http.response.status_code", http.StatusOK))

	if resp != nil {
		resp.TraceID = traceID
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

// ReplayRequest represents the parameters for triggering a JetStream stream replay.
type ReplayRequest struct {
	StartSequence uint64 `json:"start_sequence"`
	EndSequence   uint64 `json:"end_sequence"`
	FromSequence  uint64 `json:"from_sequence"` // backward-compatible alias
	ToSequence    uint64 `json:"to_sequence"`   // backward-compatible alias
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	FromTime      string `json:"from_time"` // backward-compatible alias
	ToTime        string `json:"to_time"`   // backward-compatible alias
	ReplayMode    string `json:"replay_mode"`
	ReplayFrom    string `json:"replay_from"` // "sequence" or "time"
}

// parseReplayTime parses RFC3339 or HTML datetime-local formats.
func parseReplayTime(val string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, val); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02T15:04", val); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04:05", val)
}

// ReplayJobs triggers an actual JetStream replay by configuring an ephemeral consumer.
func (h *Handler) ReplayJobs(c *gin.Context) {
	var req ReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid replay payload: %v", err)})
		return
	}

	// Handle backward-compatible aliases
	if req.StartSequence == 0 && req.FromSequence > 0 {
		req.StartSequence = req.FromSequence
	}
	if req.EndSequence == 0 && req.ToSequence > 0 {
		req.EndSequence = req.ToSequence
	}
	if req.StartTime == "" && req.FromTime != "" {
		req.StartTime = req.FromTime
	}
	if req.EndTime == "" && req.ToTime != "" {
		req.EndTime = req.ToTime
	}

	isTimeMode := req.ReplayFrom == "time" || (req.ReplayFrom == "" && req.StartTime != "")
	var parsedStartTime, parsedEndTime time.Time

	if isTimeMode {
		if req.StartTime == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start_time is required for time-based replay"})
			return
		}
		var err error
		parsedStartTime, err = parseReplayTime(req.StartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid start_time: %v", err)})
			return
		}
		if req.EndTime != "" {
			parsedEndTime, err = parseReplayTime(req.EndTime)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid end_time: %v", err)})
				return
			}
			if !parsedStartTime.Before(parsedEndTime) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "start_time must be before end_time"})
				return
			}
		}
	} else {
		// Sequence mode validation
		if req.StartSequence == 0 {
			req.StartSequence = 1
		}
		if req.EndSequence == 0 {
			req.EndSequence = req.StartSequence + 100
		}
		if req.EndSequence < req.StartSequence {
			c.JSON(http.StatusBadRequest, gin.H{"error": "end_sequence must be greater than or equal to start_sequence"})
			return
		}
	}

	js, err := h.natsClient.Conn.JetStream()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get JetStream context: %v", err)})
		return
	}

	// Verify JOBS stream exists
	_, err = js.StreamInfo("JOBS")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JOBS stream does not exist in JetStream"})
		return
	}

	consumerName := fmt.Sprintf("replay-%d", time.Now().UnixNano()%1000000)
	inbox := nats.NewInbox()

	consumerCfg := &nats.ConsumerConfig{
		Name:           consumerName,
		DeliverSubject: inbox,
		FilterSubject:  messaging.SubjectJobSubmitted,
		AckPolicy:      nats.AckNonePolicy,
	}

	if isTimeMode {
		consumerCfg.DeliverPolicy = nats.DeliverByStartTimePolicy
		consumerCfg.OptStartTime = &parsedStartTime
	} else {
		consumerCfg.DeliverPolicy = nats.DeliverByStartSequencePolicy
		consumerCfg.OptStartSeq = req.StartSequence
	}

	if req.ReplayMode == "original" {
		consumerCfg.ReplayPolicy = nats.ReplayOriginalPolicy
	} else {
		consumerCfg.ReplayPolicy = nats.ReplayInstantPolicy
	}

	cinfo, err := js.AddConsumer("JOBS", consumerCfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to create replay consumer: %v", err)})
		return
	}

	// Consume replayed messages from the delivery inbox in a background goroutine
	go func() {
		timeout := 30 * time.Second
		if req.ReplayMode == "original" {
			timeout = 60 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		defer func() {
			_ = js.DeleteConsumer("JOBS", cinfo.Name)
			log.Printf("[Replay] Teardown complete for ephemeral consumer %s", cinfo.Name)
		}()

		sub, err := h.natsClient.Conn.Subscribe(inbox, func(msg *nats.Msg) {
			meta, err := msg.Metadata()
			if err != nil {
				return
			}

			// Sequence boundary check
			if !isTimeMode && req.EndSequence > 0 && meta.Sequence.Stream > req.EndSequence {
				return
			}

			// Time boundary check
			if isTimeMode && !parsedEndTime.IsZero() && meta.Timestamp.After(parsedEndTime) {
				return
			}

			var job jobs.Job
			if err := json.Unmarshal(msg.Data, &job); err != nil {
				job.JobID = fmt.Sprintf("seq-%d", meta.Sequence.Stream)
				job.Type = "unknown"
			}

			correlationID := msg.Header.Get("X-Correlation-Id")
			if correlationID == "" {
				correlationID = fmt.Sprintf("corr-replay-%d", meta.Sequence.Stream)
			}

			eventPayload := map[string]interface{}{
				"job_id":         job.JobID,
				"type":           job.Type,
				"status":         "REPLAYED",
				"delivery_count": 1,
				"delivery_mode":  "JETSTREAM",
				"sequence":       meta.Sequence.Stream,
				"correlation_id": correlationID,
				"msg_id":         fmt.Sprintf("replay-seq-%d", meta.Sequence.Stream),
			}
			payloadBytes, _ := json.Marshal(eventPayload)

			replayMsg := nats.NewMsg(messaging.SubjectJobReplayed)
			replayMsg.Data = payloadBytes
			replayMsg.Header.Set("Content-Type", "application/json")
			replayMsg.Header.Set("X-Source", "replay-consumer")
			replayMsg.Header.Set("X-Correlation-Id", correlationID)
			replayMsg.Header.Set("X-Delivery-Mode", "JETSTREAM")
			replayMsg.Header.Set("Nats-Msg-Id", fmt.Sprintf("replay-seq-%d", meta.Sequence.Stream))

			_ = h.natsClient.Conn.PublishMsg(replayMsg)
			log.Printf("[Replay] Replayed sequence #%d (Job: %s) via %s", meta.Sequence.Stream, job.JobID, cinfo.Name)
		})
		if err != nil {
			log.Printf("[Replay] Failed to subscribe to replay inbox: %v", err)
			return
		}
		defer sub.Unsubscribe()

		<-ctx.Done()
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"status":   "REPLAY_STARTED",
		"consumer": cinfo.Name,
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

