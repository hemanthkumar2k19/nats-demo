package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"nats-demo/services/internal/jobs"
	"nats-demo/services/internal/messaging"

	"github.com/nats-io/nats.go"
)

// subscribeControlResponders registers NATS Request/Reply responders for runtime demo control.
func (a *App) subscribeControlResponders(workerName string, jobHandler messaging.JobHandler) error {
	// 1. Status Ping Responder (status.processor)
	statusSub, err := a.natsClient.Conn.Subscribe("status.processor", func(msg *nats.Msg) {
		a.mu.RLock()
		enabled := a.processingEnabled
		cType := a.consumerConfig.Type
		cName := a.consumerName
		cWorkers := a.consumerConfig.Workers
		cOrdering := a.consumerConfig.Ordering
		cDeliver := a.consumerConfig.DeliverPolicy
		cAck := a.consumerConfig.AckPolicy
		a.mu.RUnlock()

		respBytes := fmt.Appendf(nil, `{"status":"ACTIVE","processing":%t,"consumer_type":"%s","consumer_name":"%s","workers":%d,"ordering":"%s","deliver_policy":"%s","ack_policy":"%s"}`,
			enabled, cType, cName, cWorkers, cOrdering, cDeliver, cAck)
		if err := msg.Respond(respBytes); err != nil {
			log.Printf("[Processor] Failed to send status reply: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to status responder: %w", err)
	}
	a.statusSub = statusSub
	log.Println("[Run] Subscribed to status responder subject: status.processor")

	// 2. Processor State Control (processor.state.set)
	stateSetSub, err := a.natsClient.Conn.Subscribe(messaging.SubjectProcessorStateSet, func(msg *nats.Msg) {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("[Processor] Failed to unmarshal state set payload: %v", err)
			_ = msg.Respond([]byte(`{"error":"Invalid payload"}`))
			return
		}

		a.mu.Lock()
		a.processingEnabled = req.Enabled
		a.mu.Unlock()

		statusVal := "STOPPED"
		if req.Enabled {
			statusVal = "RUNNING"
			if err := a.subscribeCore(workerName, jobHandler); err != nil {
				log.Printf("[Processor] Core NATS subscription failed on toggle: %v", err)
			}
			if err := a.subscribeJetStream(); err != nil {
				log.Printf("[Processor] JetStream subscription failed on toggle: %v", err)
			}
			a.startWorkers(context.Background())
		} else {
			a.unsubscribeCore()
			a.mu.Lock()
			for _, cancel := range a.workerCancels {
				cancel()
			}
			a.workerCancels = nil
			a.mu.Unlock()
			a.unsubscribeJetStream()
		}

		log.Printf("[Processor] Processing toggled to enabled=%t (status: %s)", req.Enabled, statusVal)
		respBytes := fmt.Appendf(nil, `{"enabled":%t,"status":"%s"}`, req.Enabled, statusVal)
		if err := msg.Respond(respBytes); err != nil {
			log.Printf("[Processor] Failed to send state response: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to control subject: %w", err)
	}
	a.stateSetSub = stateSetSub
	log.Printf("[Run] Subscribed to control responder subject: %s", messaging.SubjectProcessorStateSet)

	// 5. Queue Group Configuration Control (queue.config.set)
	queueConfigSub, err := a.natsClient.Conn.Subscribe(messaging.SubjectQueueGroupConfigSet, func(msg *nats.Msg) {
		var req jobs.QueueGroupConfig
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("[Processor] Failed to unmarshal queue config payload: %v", err)
			_ = msg.Respond([]byte(`{"error":"Invalid payload"}`))
			return
		}

		if req.Workers < 1 {
			req.Workers = 1
		} else if req.Workers > 5 {
			req.Workers = 5
		}

		a.queueMu.Lock()
		a.queueWorkersCount = req.Workers
		a.queueMu.Unlock()

		if err := a.subscribeQueueGroup(); err != nil {
			log.Printf("[Processor] Failed to reconfigure queue group subscribers: %v", err)
			_ = msg.Respond(fmt.Appendf(nil, `{"error":"Failed to reconfigure: %v"}`, err))
			return
		}

		a.queueMu.Lock()
		distCopy := make(map[string]int)
		for k, v := range a.queueDistribution {
			distCopy[k] = v
		}
		resp := jobs.QueueGroupStatusResponse{
			Subject:      messaging.SubjectJobQueue,
			QueueGroup:   messaging.QueueGroupJobWorkers,
			Workers:      a.queueWorkersCount,
			Distribution: distCopy,
		}
		a.queueMu.Unlock()

		respBytes, _ := json.Marshal(resp)
		_ = msg.Respond(respBytes)
		log.Printf("[Processor] Queue Group reconfigured: Subject=%s, QueueGroup=%s, Workers=%d",
			messaging.SubjectJobQueue, messaging.QueueGroupJobWorkers, req.Workers)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to queue config subject: %w", err)
	}
	a.queueConfigSub = queueConfigSub
	log.Printf("[Run] Subscribed to queue config responder subject: %s", messaging.SubjectQueueGroupConfigSet)

	// 6. Queue Group Status Responder (queue.status)
	queueStatusSub, err := a.natsClient.Conn.Subscribe(messaging.SubjectQueueGroupStatus, func(msg *nats.Msg) {
		a.queueMu.Lock()
		distCopy := make(map[string]int)
		for k, v := range a.queueDistribution {
			distCopy[k] = v
		}
		resp := jobs.QueueGroupStatusResponse{
			Subject:      messaging.SubjectJobQueue,
			QueueGroup:   messaging.QueueGroupJobWorkers,
			Workers:      a.queueWorkersCount,
			Distribution: distCopy,
		}
		a.queueMu.Unlock()

		respBytes, _ := json.Marshal(resp)
		_ = msg.Respond(respBytes)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to queue status subject: %w", err)
	}
	a.queueStatusSub = queueStatusSub
	log.Printf("[Run] Subscribed to queue status responder subject: %s", messaging.SubjectQueueGroupStatus)

	// 7. Queue Group Distribution Reset (queue.reset)
	queueResetSub, err := a.natsClient.Conn.Subscribe(messaging.SubjectQueueGroupReset, func(msg *nats.Msg) {
		a.queueMu.Lock()
		a.queueDistribution = make(map[string]int)
		for i := 1; i <= a.queueWorkersCount; i++ {
			a.queueDistribution[fmt.Sprintf("processor-%d", i)] = 0
		}
		distCopy := make(map[string]int)
		for k, v := range a.queueDistribution {
			distCopy[k] = v
		}
		resp := jobs.QueueGroupStatusResponse{
			Subject:      messaging.SubjectJobQueue,
			QueueGroup:   messaging.QueueGroupJobWorkers,
			Workers:      a.queueWorkersCount,
			Distribution: distCopy,
		}
		a.queueMu.Unlock()

		respBytes, _ := json.Marshal(resp)
		_ = msg.Respond(respBytes)
		log.Printf("[Processor] Core NATS Queue Group distribution reset")
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to queue reset subject: %w", err)
	}
	a.queueResetSub = queueResetSub
	log.Printf("[Run] Subscribed to queue reset responder subject: %s", messaging.SubjectQueueGroupReset)

	return nil
}

// unsubscribeControlResponders deactivates demo control responders during teardown.
func (a *App) unsubscribeControlResponders() {
	if a.statusSub != nil {
		_ = a.statusSub.Unsubscribe()
		a.statusSub = nil
	}
	if a.stateSetSub != nil {
		_ = a.stateSetSub.Unsubscribe()
		a.stateSetSub = nil
	}
	if a.queueConfigSub != nil {
		_ = a.queueConfigSub.Unsubscribe()
		a.queueConfigSub = nil
	}
	if a.queueStatusSub != nil {
		_ = a.queueStatusSub.Unsubscribe()
		a.queueStatusSub = nil
	}
	if a.queueResetSub != nil {
		_ = a.queueResetSub.Unsubscribe()
		a.queueResetSub = nil
	}
}
