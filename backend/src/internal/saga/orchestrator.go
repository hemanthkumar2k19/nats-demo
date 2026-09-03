package saga

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"nats-demo/internal/messaging"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// Orchestrator coordinates 2-Operation event-driven Sagas with interactive step buttons and compensation.
type Orchestrator struct {
	nc        *nats.Conn
	mu        sync.RWMutex
	instances map[string]*SagaInstance // Keyed by JobID
	subs      []*nats.Subscription
}

// NewOrchestrator creates a new Saga Orchestrator instance.
func NewOrchestrator(nc *nats.Conn) *Orchestrator {
	o := &Orchestrator{
		nc:        nc,
		instances: make(map[string]*SagaInstance),
		subs:      make([]*nats.Subscription, 0),
	}
	o.setupEventSubscriptions()
	return o
}

func (o *Orchestrator) setupEventSubscriptions() {
	if o.nc == nil {
		return
	}

	// Listen to saga.start events
	subStart, err := o.nc.Subscribe(messaging.SubjectSagaTrigger, func(msg *nats.Msg) {
		var evt SagaEvent
		if err := json.Unmarshal(msg.Data, &evt); err == nil {
			log.Printf("[SagaOrchestrator] Event [%s] received for Job: %s", msg.Subject, evt.JobID)
		}
	})
	if err == nil {
		o.subs = append(o.subs, subStart)
	}
}

// StartSaga creates and triggers a new 2-Op Saga via NATS event.
func (o *Orchestrator) StartSaga(req StartSagaRequest) (*SagaInstance, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	jobID := req.JobID
	if jobID == "" {
		jobID = fmt.Sprintf("order-%s", uuid.New().String()[:6])
	}

	sagaID := fmt.Sprintf("saga-%s", uuid.New().String()[:6])
	now := time.Now()

	instance := &SagaInstance{
		SagaID:           sagaID,
		JobID:            jobID,
		State:            SagaStateOp1Pending,
		CurrentStep:      StepOp1Reserve,
		CompletedSteps:   make([]string, 0),
		CompensatedSteps: make([]string, 0),
		Steps:            make([]StepRecord, 0),
		FailureConfig: FailureConfig{
			FailStep:         req.FailStep,
			FailCompensation: req.FailCompensation,
			StepDelayMs:      req.StepDelayMs,
			Interactive:      req.Interactive,
		},
		CreatedAt: now,
		UpdatedAt: now,
		Payload:   req.Payload,
	}

	o.instances[jobID] = instance

	// Publish saga.start event over NATS
	o.publishEvent(messaging.SubjectSagaTrigger, instance, "2-Op Saga workflow triggered over NATS")

	// If auto-run mode (not interactive), execute in background with pacing
	if !req.Interactive {
		go o.runAutoWorkflow(instance)
	}

	log.Printf("[SagaOrchestrator] Triggered 2-Op Saga [%s] for Job [%s] (Interactive=%v)",
		sagaID, jobID, req.Interactive)
	return instance, nil
}

// AdvanceStep manually advances or fails a step in interactive mode.
func (o *Orchestrator) AdvanceStep(jobID string, req AdvanceStepRequest) (*SagaInstance, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	inst, exists := o.instances[jobID]
	if !exists {
		return nil, fmt.Errorf("saga instance for job [%s] not found", jobID)
	}

	now := time.Now()

	if req.Step == "op1" || req.Step == StepOp1Reserve {
		if inst.State != SagaStateOp1Pending && inst.State != SagaStateStarted {
			return inst, fmt.Errorf("cannot advance Op 1 in current state: %s", inst.State)
		}

		if req.Action == "SUCCESS" {
			inst.State = SagaStateOp2Pending
			inst.CurrentStep = StepOp2Payment
			inst.CompletedSteps = append(inst.CompletedSteps, StepOp1Reserve)
			inst.UpdatedAt = now
			inst.Steps = append(inst.Steps, StepRecord{
				Name:        StepOp1Reserve,
				Type:        "FORWARD",
				Status:      "SUCCESS",
				StartedAt:   now.Add(-200 * time.Millisecond),
				CompletedAt: now,
				DurationMs:  200,
				Details:     "Inventory reserved successfully",
			})
			o.publishEvent(messaging.SubjectSagaOp1Completed, inst, "Op 1 (Reserve Inventory) completed successfully")
			log.Printf("[SagaOrchestrator] Job [%s] Op 1 succeeded -> Advanced to Op 2", jobID)
		} else {
			inst.State = SagaStateFailed
			inst.CurrentStep = ""
			inst.Error = req.Error
			if inst.Error == "" {
				inst.Error = "Inventory unavailable (Out of stock)"
			}
			inst.UpdatedAt = now
			inst.Steps = append(inst.Steps, StepRecord{
				Name:        StepOp1Reserve,
				Type:        "FORWARD",
				Status:      "FAILED",
				StartedAt:   now.Add(-150 * time.Millisecond),
				CompletedAt: now,
				DurationMs:  150,
				Error:       inst.Error,
			})
			o.publishEvent(messaging.SubjectSagaOp1Failed, inst, inst.Error)
			o.publishEvent(messaging.SubjectSagaFailed, inst, "Saga failed at Op 1 (No rollback needed)")
			log.Printf("[SagaOrchestrator] Job [%s] Op 1 failed -> Saga FAILED", jobID)
		}
	} else if req.Step == "op2" || req.Step == StepOp2Payment {
		if inst.State != SagaStateOp2Pending {
			return inst, fmt.Errorf("cannot advance Op 2 in current state: %s", inst.State)
		}

		if req.Action == "SUCCESS" {
			inst.State = SagaStateCompleted
			inst.CurrentStep = ""
			inst.CompletedSteps = append(inst.CompletedSteps, StepOp2Payment)
			inst.UpdatedAt = now
			inst.Steps = append(inst.Steps, StepRecord{
				Name:        StepOp2Payment,
				Type:        "FORWARD",
				Status:      "SUCCESS",
				StartedAt:   now.Add(-250 * time.Millisecond),
				CompletedAt: now,
				DurationMs:  250,
				Details:     "Payment processed successfully",
			})
			o.publishEvent(messaging.SubjectSagaOp2Completed, inst, "Op 2 (Process Payment) completed successfully")
			o.publishEvent(messaging.SubjectSagaCompleted, inst, "2-Op Saga completed successfully")
			log.Printf("[SagaOrchestrator] Job [%s] Op 2 succeeded -> Saga COMPLETED", jobID)
		} else {
			// Op 2 failed -> Trigger Compensation for Op 1
			inst.Error = req.Error
			if inst.Error == "" {
				inst.Error = "Payment declined: insufficient funds"
			}
			inst.UpdatedAt = now
			inst.Steps = append(inst.Steps, StepRecord{
				Name:        StepOp2Payment,
				Type:        "FORWARD",
				Status:      "FAILED",
				StartedAt:   now.Add(-200 * time.Millisecond),
				CompletedAt: now,
				DurationMs:  200,
				Error:       inst.Error,
			})
			o.publishEvent(messaging.SubjectSagaOp2Failed, inst, inst.Error)

			// Initiate compensation rollback
			inst.State = SagaStateCompensating
			inst.CurrentStep = StepOp1Release
			o.publishEvent(messaging.SubjectSagaOp1Compensate, inst, "Triggering compensation: Release Inventory")
			log.Printf("[SagaOrchestrator] Job [%s] Op 2 failed -> Triggering Compensation for Op 1", jobID)

			// Execute compensation
			compStart := time.Now()
			if req.FailCompensation {
				inst.State = SagaStateCompensationFailed
				inst.Error = "Compensation failed: unable to release inventory hold"
				inst.UpdatedAt = time.Now()
				inst.Steps = append(inst.Steps, StepRecord{
					Name:        StepOp1Release,
					Type:        "COMPENSATION",
					Status:      "FAILED",
					StartedAt:   compStart,
					CompletedAt: time.Now(),
					DurationMs:  180,
					Error:       "Release inventory failed",
				})
				o.publishEvent(messaging.SubjectSagaFailed, inst, "Saga compensation failed: manual intervention required")
			} else {
				inst.State = SagaStateFailed
				inst.CompensatedSteps = append(inst.CompensatedSteps, StepOp1Reserve)
				inst.UpdatedAt = time.Now()
				inst.Steps = append(inst.Steps, StepRecord{
					Name:        StepOp1Release,
					Type:        "COMPENSATION",
					Status:      "SUCCESS",
					StartedAt:   compStart,
					CompletedAt: time.Now(),
					DurationMs:  180,
					Details:     "Inventory hold successfully released and refunded",
				})
				o.publishEvent(messaging.SubjectSagaOp1Compensated, inst, "Op 1 hold released")
				o.publishEvent(messaging.SubjectSagaFailed, inst, "Saga failed: rollback compensation completed")
			}
			inst.CurrentStep = ""
		}
	}

	cpy := *inst
	return &cpy, nil
}

func (o *Orchestrator) runAutoWorkflow(inst *SagaInstance) {
	delay := inst.FailureConfig.StepDelayMs
	if delay <= 0 {
		delay = 800
	}

	time.Sleep(time.Duration(delay) * time.Millisecond)

	// Step 1
	action1 := "SUCCESS"
	if inst.FailureConfig.FailStep == "op1" || inst.FailureConfig.FailStep == StepOp1Reserve {
		action1 = "FAIL"
	}
	_, _ = o.AdvanceStep(inst.JobID, AdvanceStepRequest{
		Step:   "op1",
		Action: action1,
	})

	if action1 == "FAIL" {
		return
	}

	time.Sleep(time.Duration(delay) * time.Millisecond)

	// Step 2
	action2 := "SUCCESS"
	if inst.FailureConfig.FailStep == "op2" || inst.FailureConfig.FailStep == StepOp2Payment {
		action2 = "FAIL"
	}
	_, _ = o.AdvanceStep(inst.JobID, AdvanceStepRequest{
		Step:             "op2",
		Action:           action2,
		FailCompensation: inst.FailureConfig.FailCompensation,
	})
}

// GetSaga retrieves a Saga instance by JobID.
func (o *Orchestrator) GetSaga(jobID string) (*SagaInstance, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	inst, exists := o.instances[jobID]
	if !exists {
		return nil, false
	}
	cpy := *inst
	return &cpy, true
}

// ListSagas returns all tracked Saga instances.
func (o *Orchestrator) ListSagas() []*SagaInstance {
	o.mu.RLock()
	defer o.mu.RUnlock()

	list := make([]*SagaInstance, 0, len(o.instances))
	for _, inst := range o.instances {
		cpy := *inst
		list = append(list, &cpy)
	}
	return list
}

func (o *Orchestrator) publishEvent(subject string, inst *SagaInstance, msg string) {
	if o.nc == nil {
		return
	}

	evt := SagaEvent{
		SagaID:    inst.SagaID,
		JobID:     inst.JobID,
		EventType: subject,
		State:     inst.State,
		Step:      inst.CurrentStep,
		Timestamp: time.Now(),
		Message:   msg,
	}

	data, err := json.Marshal(evt)
	if err == nil {
		_ = o.nc.Publish(subject, data)
	}
}
