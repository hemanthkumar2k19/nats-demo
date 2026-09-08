package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"nats-demo/services/internal/jobs"
	"nats-demo/services/internal/messaging"

	"github.com/nats-io/nats.go"
)

// subscribeQueueGroup configures Core NATS Queue Group subscriptions based on a.queueWorkersCount.
func (a *App) subscribeQueueGroup() error {
	a.queueMu.Lock()
	defer a.queueMu.Unlock()

	for _, sub := range a.queueSubs {
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}
	a.queueSubs = make([]*nats.Subscription, 0, a.queueWorkersCount)

	for i := 1; i <= a.queueWorkersCount; i++ {
		workerName := fmt.Sprintf("processor-%d", i)
		wName := workerName
		sub, err := a.natsClient.Conn.QueueSubscribe(messaging.SubjectJobQueue, messaging.QueueGroupJobWorkers, func(msg *nats.Msg) {
			a.handleQueueMessage(wName, msg)
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe %s to queue group %s: %w", wName, messaging.QueueGroupJobWorkers, err)
		}
		a.queueSubs = append(a.queueSubs, sub)
		log.Printf("[%s] Subscribed to Core NATS queue group '%s' on subject '%s'",
			wName, messaging.QueueGroupJobWorkers, messaging.SubjectJobQueue)
	}

	return nil
}

// unsubscribeQueueGroup cleans up active Core NATS queue group subscriptions.
func (a *App) unsubscribeQueueGroup() {
	a.queueMu.Lock()
	defer a.queueMu.Unlock()

	if len(a.queueSubs) == 0 {
		return
	}

	for _, sub := range a.queueSubs {
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}
	a.queueSubs = nil
	log.Println("[Processor] Core NATS queue group subscriptions deactivated")
}

// handleQueueMessage processes a message delivered by Core NATS to a queue group member.
func (a *App) handleQueueMessage(workerName string, msg *nats.Msg) {
	var job jobs.Job
	if err := json.Unmarshal(msg.Data, &job); err != nil {
		log.Printf("[%s] [queue-group: %s] Failed to unmarshal message: %v",
			workerName, messaging.QueueGroupJobWorkers, err)
		return
	}

	a.queueMu.Lock()
	a.queueDistribution[workerName]++
	currCount := a.queueDistribution[workerName]
	a.queueMu.Unlock()

	log.Printf("[%s] [queue-group: %s] Received job %s (worker total: %d)",
		workerName, messaging.QueueGroupJobWorkers, job.JobID, currCount)

	// Publish RECEIVED lifecycle event to jobs.queue.received with worker name
	_ = a.publisher.PublishJobLifecycle(
		messaging.SubjectJobQueueReceived,
		job.JobID,
		"RECEIVED",
		1,
		"",
		workerName,
		"CORE",
		0,
	)

	// Simulate processing time
	time.Sleep(150 * time.Millisecond)

	// Publish COMPLETED lifecycle event to jobs.queue.completed with worker name
	_ = a.publisher.PublishJobLifecycle(
		messaging.SubjectJobQueueCompleted,
		job.JobID,
		"COMPLETED",
		1,
		"",
		workerName,
		"CORE",
		0,
	)
}
