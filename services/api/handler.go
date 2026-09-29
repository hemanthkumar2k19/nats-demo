package api

import (
	"encoding/json"
	"net/http"
	"services/publisher"
	"services/telemetry"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Handler provides HTTP handlers with trace propagation.
type Handler struct {
	pubService *publisher.Service
}

// NewHandler constructs a new API Handler instance.
func NewHandler(pubService *publisher.Service) *Handler {
	return &Handler{pubService: pubService}
}

// PublishHandler handles HTTP POST /api/v1/publish requests, extracting W3C trace context.
func (h *Handler) PublishHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Extract W3C trace context from incoming HTTP headers
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	// 2. Start HTTP server span using unified tracer instance
	ctx, span := telemetry.Tracer().Start(ctx, "HTTP POST "+r.URL.Path, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	// 3. Decode request body
	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Subject == "" {
		http.Error(w, "Field 'subject' is required", http.StatusBadRequest)
		return
	}

	// 4. Delegate to Service Layer (passing active context with HTTP server span)
	if err := h.pubService.PublishEvent(ctx, req.Subject, []byte(req.Data)); err != nil {
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(PublishResponse{
		Status:  "published",
		Subject: req.Subject,
	})
}
