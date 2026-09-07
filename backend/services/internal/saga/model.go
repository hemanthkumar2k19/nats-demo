package saga

import (
	"time"
)

// Saga state constants
type SagaState string

const (
	SagaStateStarted            SagaState = "STARTED"
	SagaStateOp1Pending         SagaState = "OP1_PENDING"
	SagaStateOp1Completed       SagaState = "OP1_COMPLETED"
	SagaStateOp2Pending         SagaState = "OP2_PENDING"
	SagaStateCompensating       SagaState = "COMPENSATING"
	SagaStateCompleted          SagaState = "COMPLETED"
	SagaStateFailed             SagaState = "FAILED"
	SagaStateCompensationFailed SagaState = "COMPENSATION_FAILED"
	SagaStateCancelled          SagaState = "CANCELLED"
)

// 2-Operation Step Names
const (
	StepOp1Reserve = "reserve"
	StepOp2Payment = "payment"
	StepOp1Release = "release"
)

// Worker Command Step Names
const (
	StepAllocate = "allocate"
	StepPrepare  = "prepare"
	StepExecute  = "execute"
	StepRelease  = "release"
)

// StepRecord stores the execution result of an individual saga step.
type StepRecord struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"` // "FORWARD" or "COMPENSATION"
	Status      string    `json:"status"` // "PENDING", "RUNNING", "SUCCESS", "FAILED"
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	DurationMs  int64     `json:"duration_ms"`
	Error       string    `json:"error,omitempty"`
	Details     string    `json:"details,omitempty"`
}

// FailureConfig configures controlled simulation of step or compensation failure.
type FailureConfig struct {
	FailStep         string `json:"fail_step,omitempty"`          // "op1", "op2", "reserve", "payment"
	FailCompensation bool   `json:"fail_compensation,omitempty"` // Fail during rollback
	StepDelayMs      int    `json:"step_delay_ms,omitempty"`     // Delay per step for auto-run
	Interactive      bool   `json:"interactive"`                 // Wait for explicit UI button click per step
}

// SagaInstance represents a running or completed Saga workflow.
type SagaInstance struct {
	SagaID           string                 `json:"saga_id"`
	JobID            string                 `json:"job_id"`
	State            SagaState              `json:"state"`
	CurrentStep      string                 `json:"current_step,omitempty"`
	CompletedSteps   []string               `json:"completed_steps"`
	CompensatedSteps []string               `json:"compensated_steps"`
	Steps            []StepRecord           `json:"steps"`
	FailureConfig    FailureConfig          `json:"failure_config"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	Error            string                 `json:"error,omitempty"`
	Payload          map[string]interface{} `json:"payload,omitempty"`
}

// StartSagaRequest defines the payload to initiate a 2-Op Saga.
type StartSagaRequest struct {
	JobID            string                 `json:"job_id"`
	Type             string                 `json:"type"`
	Payload          map[string]interface{} `json:"payload"`
	FailStep         string                 `json:"fail_step,omitempty"`
	FailCompensation bool                   `json:"fail_compensation,omitempty"`
	StepDelayMs      int                    `json:"step_delay_ms,omitempty"`
	Interactive      bool                   `json:"interactive"` // Manual button advance mode
}

// AdvanceStepRequest allows the UI to manually advance or fail a specific stage.
type AdvanceStepRequest struct {
	Step             string `json:"step"`              // "op1" or "op2"
	Action           string `json:"action"`            // "SUCCESS" or "FAIL"
	Error            string `json:"error,omitempty"`   // Optional error description
	FailCompensation bool   `json:"fail_compensation"` // If failing rollback
}

// InjectFailureRequest modifies failure behavior on an active Saga.
type InjectFailureRequest struct {
	FailStep         string `json:"fail_step"`
	FailCompensation bool   `json:"fail_compensation"`
}

// SagaEvent represents an event-driven broadcast published on saga.*
type SagaEvent struct {
	SagaID    string    `json:"saga_id"`
	JobID     string    `json:"job_id"`
	EventType string    `json:"event_type"`
	State     SagaState `json:"state"`
	Step      string    `json:"step,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

// SagaCommandPayload defines the command payload sent to worker responders.
type SagaCommandPayload struct {
	SagaID           string                 `json:"saga_id"`
	JobID            string                 `json:"job_id"`
	Step             string                 `json:"step"`
	StepDelayMs      int                    `json:"step_delay_ms,omitempty"`
	FailStep         string                 `json:"fail_step,omitempty"`
	FailCompensation bool                   `json:"fail_compensation,omitempty"`
	Payload          map[string]interface{} `json:"payload,omitempty"`
}

// SagaCommandResponse defines the response returned by worker responders.
type SagaCommandResponse struct {
	Success bool   `json:"success"`
	Step    string `json:"step"`
	Details string `json:"details,omitempty"`
	Error   string `json:"error,omitempty"`
}

