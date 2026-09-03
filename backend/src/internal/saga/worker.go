package saga

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"nats-demo/internal/messaging"

	"github.com/nats-io/nats.go"
)

// WorkerResponders handles command subscriptions for Saga operations in processor-service.
type WorkerResponders struct {
	nc   *nats.Conn
	subs []*nats.Subscription
}

// NewWorkerResponders creates and registers command responders.
func NewWorkerResponders(nc *nats.Conn) *WorkerResponders {
	return &WorkerResponders{
		nc: nc,
	}
}

// Start registers subscriptions on all saga.* command subjects.
func (w *WorkerResponders) Start() error {
	commands := []struct {
		subject string
		step    string
		handler func(msg *nats.Msg)
	}{
		{messaging.SubjectSagaJobAllocate, StepAllocate, w.handleAllocate},
		{messaging.SubjectSagaJobPrepare, StepPrepare, w.handlePrepare},
		{messaging.SubjectSagaJobExecute, StepExecute, w.handleExecute},
		{messaging.SubjectSagaJobRelease, StepRelease, w.handleRelease},
	}

	for _, cmd := range commands {
		sub, err := w.nc.Subscribe(cmd.subject, cmd.handler)
		if err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", cmd.subject, err)
		}
		w.subs = append(w.subs, sub)
		log.Printf("[SagaWorker] Registered responder on %s", cmd.subject)
	}

	return nil
}

// Stop unregisters all command responders.
func (w *WorkerResponders) Stop() {
	for _, sub := range w.subs {
		_ = sub.Unsubscribe()
	}
	w.subs = nil
}

func (w *WorkerResponders) handleAllocate(msg *nats.Msg) {
	w.processStep(msg, StepAllocate, false)
}

func (w *WorkerResponders) handlePrepare(msg *nats.Msg) {
	w.processStep(msg, StepPrepare, false)
}

func (w *WorkerResponders) handleExecute(msg *nats.Msg) {
	w.processStep(msg, StepExecute, false)
}

func (w *WorkerResponders) handleRelease(msg *nats.Msg) {
	w.processStep(msg, StepRelease, true)
}

func (w *WorkerResponders) processStep(msg *nats.Msg, stepName string, isCompensation bool) {
	var cmd SagaCommandPayload
	if err := json.Unmarshal(msg.Data, &cmd); err != nil {
		log.Printf("[SagaWorker] Failed to unmarshal %s command: %v", stepName, err)
		resp := SagaCommandResponse{
			Success: false,
			Step:    stepName,
			Error:   fmt.Sprintf("invalid command payload: %v", err),
		}
		data, _ := json.Marshal(resp)
		_ = msg.Respond(data)
		return
	}

	// Apply artificial step delay so user can observe step progression in UI
	delayMs := cmd.StepDelayMs
	if delayMs <= 0 {
		delayMs = 600 // Default 600ms per step for clear demo visualization
	}
	time.Sleep(time.Duration(delayMs) * time.Millisecond)

	// Evaluate forward failure condition
	if !isCompensation && cmd.FailStep == stepName {
		log.Printf("[SagaWorker] Injected failure triggered at step [%s] for Saga ID: %s", stepName, cmd.SagaID)
		resp := SagaCommandResponse{
			Success: false,
			Step:    stepName,
			Error:   fmt.Sprintf("simulated failure at [%s] step", stepName),
			Details: "Controlled failure injected for Saga demonstration",
		}
		data, _ := json.Marshal(resp)
		_ = msg.Respond(data)
		return
	}

	// Evaluate compensation failure condition
	if isCompensation && cmd.FailCompensation {
		log.Printf("[SagaWorker] Injected compensation failure triggered during [%s] for Saga ID: %s", stepName, cmd.SagaID)
		resp := SagaCommandResponse{
			Success: false,
			Step:    stepName,
			Error:   "simulated compensation failure: failed to release allocated resource",
			Details: "Controlled compensation failure injected for Saga demonstration",
		}
		data, _ := json.Marshal(resp)
		_ = msg.Respond(data)
		return
	}

	// Success response
	details := fmt.Sprintf("[%s] step completed successfully", stepName)
	if isCompensation {
		details = "Resource successfully released and cleaned up"
	}
	log.Printf("[SagaWorker] Step [%s] succeeded for Saga ID: %s", stepName, cmd.SagaID)

	resp := SagaCommandResponse{
		Success: true,
		Step:    stepName,
		Details: details,
	}
	data, _ := json.Marshal(resp)
	_ = msg.Respond(data)
}
