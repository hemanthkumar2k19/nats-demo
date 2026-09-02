package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"nats-demo/internal/activity"
	"nats-demo/internal/jobs"
	"nats-demo/internal/messaging"
	"nats-demo/internal/natsclient"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

// ControlHandler handles HTTP requests for demo inspection and UI controls in demo-control-service.
type ControlHandler struct {
	activityTracker *activity.Tracker
	natsClient      *natsclient.Client
	observer        *messaging.Observer
	jobServiceURL   string
}

// NewControlHandler instantiates a new ControlHandler.
func NewControlHandler(activityTracker *activity.Tracker, natsClient *natsclient.Client, observer *messaging.Observer, jobServiceURL string) *ControlHandler {
	if jobServiceURL == "" {
		jobServiceURL = "http://localhost:8081"
	}
	return &ControlHandler{
		activityTracker: activityTracker,
		natsClient:      natsClient,
		observer:        observer,
		jobServiceURL:   jobServiceURL,
	}
}

// GetStatus checks and returns status of NATS server, job-service, processor-service, and JetStream.
func (h *ControlHandler) GetStatus(c *gin.Context) {
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

	// Check job-service health via HTTP ping
	jobServiceStatus := "OFFLINE"
	client := http.Client{Timeout: 300 * time.Millisecond}
	resp, err := client.Get(h.jobServiceURL + "/health")
	if err == nil && resp != nil {
		if resp.StatusCode == http.StatusOK {
			jobServiceStatus = "ACTIVE"
		}
		_ = resp.Body.Close()
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
				"name":   "demo-control-service",
				"status": "ACTIVE",
			},
			{
				"name":   "job-service",
				"status": jobServiceStatus,
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

// GetActivities returns the activity events list captured from NATS.
func (h *ControlHandler) GetActivities(c *gin.Context) {
	activities := h.activityTracker.GetActivities()
	c.JSON(http.StatusOK, activities)
}

// parseReplayTime attempts to parse a time string using common ISO-8601 layouts.
func parseReplayTime(val string) (time.Time, error) {
	if val == "" {
		return time.Time{}, errors.New("empty time string")
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		time.RFC3339Nano,
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, val); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse time: %s", val)
}

// ReplayJobs triggers an ephemeral JetStream replay consumer.
func (h *ControlHandler) ReplayJobs(c *gin.Context) {
	var req struct {
		ReplayFrom    string `json:"replay_from"`
		StartSequence uint64 `json:"start_sequence"`
		EndSequence   uint64 `json:"end_sequence"`
		FromSequence  uint64 `json:"from_sequence"`
		ToSequence    uint64 `json:"to_sequence"`
		StartTime     string `json:"start_time"`
		EndTime       string `json:"end_time"`
		FromTime      string `json:"from_time"`
		ToTime        string `json:"to_time"`
		ReplayMode    string `json:"replay_mode"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid replay payload: " + err.Error()})
		return
	}

	// Normalise legacy fields
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
	if req.ReplayFrom == "" {
		if req.StartTime != "" {
			req.ReplayFrom = "time"
		} else {
			req.ReplayFrom = "sequence"
		}
	}

	isTimeMode := req.ReplayFrom == "time"
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

	// Consume replayed messages from delivery inbox in a background routine
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

			if !isTimeMode && req.EndSequence > 0 && meta.Sequence.Stream > req.EndSequence {
				return
			}
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
func (h *ControlHandler) GetSubscriptions(c *gin.Context) {
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
func (h *ControlHandler) GetAddressingActivity(c *gin.Context) {
	events := h.observer.GetEvents()
	c.JSON(http.StatusOK, gin.H{
		"events": events,
	})
}

// PutProcessorState toggles processor background processing state.
func (h *ControlHandler) PutProcessorState(c *gin.Context) {
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

	reply, err := h.natsClient.Conn.Request(messaging.SubjectProcessorStateSet, payload, 2*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Processor service did not respond: " + err.Error()})
		return
	}

	var resp struct {
		Status  string `json:"status"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid response from processor"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetConsumerStatus returns active consumer metrics and configuration.
func (h *ControlHandler) GetConsumerStatus(c *gin.Context) {
	natsStatus := "DISCONNECTED"
	if h.natsClient != nil && h.natsClient.Conn != nil && h.natsClient.Conn.Status() == nats.CONNECTED {
		natsStatus = "CONNECTED"
	}

	consumerName := "job-processor"
	consumerType := "durable"
	workers := 1
	ordering := "normal"

	if natsStatus == "CONNECTED" {
		reply, err := h.natsClient.Conn.Request("status.processor", nil, 500*time.Millisecond)
		if err == nil && len(reply.Data) > 0 {
			var procStatus struct {
				Status       string `json:"status"`
				Processing   bool   `json:"processing"`
				Workers      int    `json:"workers"`
				ConsumerName string `json:"consumer_name"`
				ConsumerType string `json:"consumer_type"`
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
			}
		}
	}

	var pending, ackPending, redelivered int
	if natsStatus == "CONNECTED" {
		js, err := h.natsClient.Conn.JetStream()
		if err == nil {
			cinfo, err := js.ConsumerInfo("JOBS", consumerName)
			if err != nil && consumerName != "processor-durable" {
				cinfo, err = js.ConsumerInfo("JOBS", "processor-durable")
			}
			if err == nil && cinfo != nil {
				pending = int(cinfo.NumPending)
				ackPending = cinfo.NumAckPending
				redelivered = cinfo.NumRedelivered
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"name":        consumerName,
		"type":        consumerType,
		"workers":     workers,
		"ordering":    ordering,
		"delivery":    "PUSH",
		"status":      "ACTIVE",
		"pending":     pending,
		"ack_pending": ackPending,
		"redelivered": redelivered,
	})
}

// PutConsumerConfig forwards consumer configuration changes to processor-service over NATS.
func (h *ControlHandler) PutConsumerConfig(c *gin.Context) {
	var req jobs.ConsumerConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid consumer configuration payload: " + err.Error()})
		return
	}

	if req.Type != "durable" && req.Type != "ephemeral" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Consumer type must be 'durable' or 'ephemeral'"})
		return
	}
	if req.Workers < 1 || req.Workers > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workers must be 1 or 2"})
		return
	}
	if req.Ordering != "normal" && req.Ordering != "ordered" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ordering must be 'normal' or 'ordered'"})
		return
	}

	payload, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal consumer config"})
		return
	}

	reply, err := h.natsClient.Conn.Request(messaging.SubjectConsumerConfigSet, payload, 3*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Processor service did not respond to consumer update: " + err.Error()})
		return
	}

	var resp struct {
		Status       string `json:"status"`
		ConsumerName string `json:"consumer_name"`
		Type         string `json:"type"`
		Workers      int    `json:"workers"`
		Ordering     string `json:"ordering"`
		Error        string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid response from processor"})
		return
	}
	if resp.Error != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": resp.Error})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DLQMessage represents a failed message in the JOBS_DLQ stream.
type DLQMessage struct {
	Sequence         uint64         `json:"sequence"`
	JobID            string         `json:"job_id"`
	Type             string         `json:"type"`
	OriginalSubject  string         `json:"original_subject"`
	DeliveryAttempts int            `json:"delivery_attempts"`
	FailureReason    string         `json:"failure_reason"`
	Timestamp        string         `json:"timestamp"`
	CorrelationID    string         `json:"correlation_id"`
	Worker           string         `json:"worker,omitempty"`
	Payload          map[string]any `json:"payload,omitempty"`
}

// GetDLQStatus returns stream metrics for JOBS_DLQ and consumer stats for dlq-inspector.
func (h *ControlHandler) GetDLQStatus(c *gin.Context) {
	if h.natsClient == nil || h.natsClient.Conn == nil || h.natsClient.Conn.Status() != nats.CONNECTED {
		c.JSON(http.StatusOK, gin.H{
			"stream":   "JOBS_DLQ",
			"messages": 0,
			"bytes":    0,
			"consumer": "dlq-inspector",
			"pending":  0,
		})
		return
	}

	js, err := h.natsClient.Conn.JetStream()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get JetStream context"})
		return
	}

	var totalMsgs uint64
	var totalBytes uint64
	var firstSeq, lastSeq uint64
	sinfo, err := js.StreamInfo("JOBS_DLQ")
	if err == nil && sinfo != nil {
		totalMsgs = sinfo.State.Msgs
		totalBytes = sinfo.State.Bytes
		firstSeq = sinfo.State.FirstSeq
		lastSeq = sinfo.State.LastSeq
	}

	var pending uint64 = totalMsgs
	var ackPending int
	cinfo, err := js.ConsumerInfo("JOBS_DLQ", "dlq-inspector")
	if err == nil && cinfo != nil {
		pending = cinfo.NumPending
		ackPending = cinfo.NumAckPending
	}

	c.JSON(http.StatusOK, gin.H{
		"stream":      "JOBS_DLQ",
		"messages":    totalMsgs,
		"bytes":       totalBytes,
		"first_seq":   firstSeq,
		"last_seq":    lastSeq,
		"consumer":    "dlq-inspector",
		"pending":     pending,
		"ack_pending": ackPending,
	})
}

// GetDLQMessages returns all failed messages persisted in JOBS_DLQ.
func (h *ControlHandler) GetDLQMessages(c *gin.Context) {
	if h.natsClient == nil || h.natsClient.Conn == nil || h.natsClient.Conn.Status() != nats.CONNECTED {
		c.JSON(http.StatusOK, []DLQMessage{})
		return
	}

	js, err := h.natsClient.Conn.JetStream()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get JetStream context"})
		return
	}

	sinfo, err := js.StreamInfo("JOBS_DLQ")
	if err != nil || sinfo == nil || sinfo.State.Msgs == 0 {
		c.JSON(http.StatusOK, []DLQMessage{})
		return
	}

	messages := make([]DLQMessage, 0, sinfo.State.Msgs)
	for seq := sinfo.State.FirstSeq; seq <= sinfo.State.LastSeq; seq++ {
		rawMsg, err := js.GetMsg("JOBS_DLQ", seq)
		if err != nil || rawMsg == nil {
			continue
		}

		var item DLQMessage
		if err := json.Unmarshal(rawMsg.Data, &item); err == nil {
			item.Sequence = seq
			if item.Timestamp == "" {
				item.Timestamp = rawMsg.Time.UTC().Format(time.RFC3339)
			}
			if item.OriginalSubject == "" {
				item.OriginalSubject = rawMsg.Header.Get("X-Original-Subject")
			}
			if item.CorrelationID == "" {
				item.CorrelationID = rawMsg.Header.Get("X-Correlation-Id")
			}
			messages = append(messages, item)
		}
	}

	c.JSON(http.StatusOK, messages)
}
