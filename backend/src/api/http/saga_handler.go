package http

import (
	"net/http"

	"nats-demo/internal/saga"

	"github.com/gin-gonic/gin"
)

// SagaHandler handles HTTP endpoints for the Saga Orchestrator.
type SagaHandler struct {
	orchestrator *saga.Orchestrator
}

// NewSagaHandler creates a new SagaHandler.
func NewSagaHandler(orchestrator *saga.Orchestrator) *SagaHandler {
	return &SagaHandler{
		orchestrator: orchestrator,
	}
}

// StartSaga starts a new Saga workflow.
// POST /sagas/jobs
func (h *SagaHandler) StartSaga(c *gin.Context) {
	var req saga.StartSagaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instance, err := h.orchestrator.StartSaga(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, instance)
}

// AdvanceStep allows manual step progression or failure in interactive mode.
// POST /sagas/jobs/:job_id/step
func (h *SagaHandler) AdvanceStep(c *gin.Context) {
	jobID := c.Param("job_id")
	var req saga.AdvanceStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instance, err := h.orchestrator.AdvanceStep(jobID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, instance)
}

// GetSagaStatus returns the current status and step history of a Saga.
// GET /sagas/jobs/:job_id
func (h *SagaHandler) GetSagaStatus(c *gin.Context) {
	jobID := c.Param("job_id")
	instance, exists := h.orchestrator.GetSaga(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "saga instance not found"})
		return
	}

	c.JSON(http.StatusOK, instance)
}

// ListSagas returns all tracked Saga instances.
// GET /sagas/jobs
func (h *SagaHandler) ListSagas(c *gin.Context) {
	list := h.orchestrator.ListSagas()
	c.JSON(http.StatusOK, list)
}
