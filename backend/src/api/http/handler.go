package http

// Handler provides backward-compatible access to JobHandler and ControlHandler.
type Handler struct {
	*JobHandler
	*ControlHandler
}

// NewHandler creates a backward-compatible composite handler.
func NewHandler(jobHandler *JobHandler, controlHandler *ControlHandler) *Handler {
	return &Handler{
		JobHandler:     jobHandler,
		ControlHandler: controlHandler,
	}
}
