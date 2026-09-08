package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

// ProcessorEvent represents a single execution lifecycle event in the processor worker.
type ProcessorEvent struct {
	JobID        string `json:"job_id"`
	JobType      string `json:"job_type"`
	Tag          string `json:"tag"`
	TagColor     string `json:"tag_color"`
	Details      string `json:"details,omitempty"`
	Timestamp    string `json:"timestamp"`
	DeliveryMode string `json:"delivery_mode"`
	Attempt      int    `json:"attempt"`
}

// EventTracker stores a thread-safe ring buffer of recent processor events.
type EventTracker struct {
	mu     sync.RWMutex
	events []ProcessorEvent
	maxCap int
}

// NewEventTracker initializes a new in-memory event tracker.
func NewEventTracker(maxCap int) *EventTracker {
	if maxCap <= 0 {
		maxCap = 50
	}
	return &EventTracker{
		events: make([]ProcessorEvent, 0, maxCap),
		maxCap: maxCap,
	}
}

// Record appends a new event to the tracker, maintaining the max cap.
func (et *EventTracker) Record(event ProcessorEvent) {
	et.mu.Lock()
	defer et.mu.Unlock()

	// Prepend event so newest appears first
	et.events = append([]ProcessorEvent{event}, et.events...)
	if len(et.events) > et.maxCap {
		et.events = et.events[:et.maxCap]
	}
}

// GetEvents returns a copy of all recorded events.
func (et *EventTracker) GetEvents() []ProcessorEvent {
	et.mu.RLock()
	defer et.mu.RUnlock()

	result := make([]ProcessorEvent, len(et.events))
	copy(result, et.events)
	return result
}

// Clear removes all recorded events.
func (et *EventTracker) Clear() {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.events = make([]ProcessorEvent, 0, et.maxCap)
}

// recordWorkerEvent is a convenience helper on App to record worker lifecycle steps.
func (a *App) recordWorkerEvent(jobID, jobType, tag, tagColor, details string, attempt int, deliveryMode string) {
	if a.eventTracker == nil {
		return
	}
	if deliveryMode == "" {
		deliveryMode = "JETSTREAM"
	}
	event := ProcessorEvent{
		JobID:        jobID,
		JobType:      jobType,
		Tag:          tag,
		TagColor:     tagColor,
		Details:      details,
		Timestamp:    time.Now().Format("15:04:05"),
		DeliveryMode: deliveryMode,
		Attempt:      attempt,
	}
	a.eventTracker.Record(event)
}

// startHTTPServer starts the dedicated processor HTTP API server.
func (a *App) startHTTPServer(port string) *http.Server {
	if port == "" {
		port = "8082"
	}

	mux := http.NewServeMux()

	// GET /processor/events - returns recent processor execution events
	// DELETE /processor/events - clears the event buffer
	mux.HandleFunc("/processor/events", func(w http.ResponseWriter, r *http.Request) {
		setCorsHeaders(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodDelete {
			a.eventTracker.Clear()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})
			return
		}

		if r.Method == http.MethodGet {
			events := a.eventTracker.GetEvents()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(events)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// GET /processor/status - returns worker execution and pull loop state
	mux.HandleFunc("/processor/status", func(w http.ResponseWriter, r *http.Request) {
		setCorsHeaders(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodGet {
			a.mu.RLock()
			processing := a.processingEnabled
			consumerName := a.consumerName
			workers := a.consumerConfig.Workers
			gStatus := a.goroutineStatus
			activeGoroutines := a.activeGoroutines
			crashedWorker := a.crashedWorker
			jsConsumer := a.jsConsumer
			a.mu.RUnlock()

			a.scenarioMu.RLock()
			activeScenario := a.activeScenario
			a.scenarioMu.RUnlock()

			var ackFloor, deliveredSeq uint64
			if jsConsumer != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				if info, err := jsConsumer.Info(ctx); err == nil && info != nil {
					ackFloor = info.AckFloor.Stream
					deliveredSeq = info.Delivered.Stream
				}
				cancel()
			}

			if workers <= 0 {
				workers = 1
			}
			if gStatus == "" {
				if processing {
					gStatus = "RUNNING"
				} else {
					gStatus = "PAUSED"
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":            "active",
				"processing":        processing,
				"consumer":          consumerName,
				"stream":            "JOBS",
				"workers":           workers,
				"scenario":          activeScenario,
				"goroutine_status":  gStatus,
				"active_goroutines": activeGoroutines,
				"crashed_worker":    crashedWorker,
				"ack_wait_seconds":  5,
				"ack_policy":        "explicit",
				"ack_floor":         ackFloor,
				"delivered_seq":     deliveredSeq,
			})
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// PUT /processor/workers - dynamically scales worker pool size (1-5)
	mux.HandleFunc("/processor/workers", func(w http.ResponseWriter, r *http.Request) {
		setCorsHeaders(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodPut {
			var body struct {
				Workers int `json:"workers"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
				return
			}

			if body.Workers < 1 {
				body.Workers = 1
			} else if body.Workers > 5 {
				body.Workers = 5
			}

			a.ScaleWorkers(body.Workers)

			a.mu.RLock()
			workers := a.consumerConfig.Workers
			active := a.activeGoroutines
			a.mu.RUnlock()

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":            "scaled",
				"workers":           workers,
				"active_goroutines": active,
			})
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// PUT /processor/scenario - arms or updates failure scenario on worker
	mux.HandleFunc("/processor/scenario", func(w http.ResponseWriter, r *http.Request) {
		setCorsHeaders(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodPut {
			var body struct {
				Scenario string `json:"scenario"`
				Once     bool   `json:"once"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
				return
			}

			if body.Scenario == "" {
				body.Scenario = "normal"
			}

			a.scenarioMu.Lock()
			a.activeScenario = body.Scenario
			a.scenarioOnce = body.Once
			a.scenarioMu.Unlock()

			log.Printf("[HTTP] Worker failure scenario set: scenario=%s, once=%t", body.Scenario, body.Once)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":   "ok",
				"scenario": body.Scenario,
				"once":     body.Once,
			})
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// POST /processor/restart - manually respawns worker goroutines
	mux.HandleFunc("/processor/restart", func(w http.ResponseWriter, r *http.Request) {
		setCorsHeaders(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodPost {
			a.mu.Lock()
			a.goroutineStatus = "RUNNING"
			a.activeGoroutines = 1
			a.mu.Unlock()

			log.Println("[HTTP] Manual supervisor restart triggered for worker goroutines")
			a.recordWorkerEvent("SYSTEM", "SUPERVISOR", "[SUPERVISOR RESPAWN]", "#8B5CF6", "Manual supervisor restart of worker goroutine", 0, "JETSTREAM")

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "restarted",
			})
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// PUT /processor/state - direct pause/resume toggle without using NATS
	mux.HandleFunc("/processor/state", func(w http.ResponseWriter, r *http.Request) {
		setCorsHeaders(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodPut {
			var body struct {
				Enabled bool `json:"enabled"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
				return
			}

			a.mu.Lock()
			a.processingEnabled = body.Enabled
			if body.Enabled {
				a.goroutineStatus = "RUNNING"
				a.activeGoroutines = 1
			} else {
				a.goroutineStatus = "PAUSED"
				a.activeGoroutines = 0
			}
			a.mu.Unlock()

			stateStr := "ON"
			if !body.Enabled {
				stateStr = "OFF"
			}
			log.Printf("[HTTP] Processor state updated directly to %s", stateStr)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     stateStr,
				"processing": body.Enabled,
			})
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("[HTTP] Processor HTTP server listening on http://localhost:%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[HTTP] Server error: %v", err)
		}
	}()

	return server
}

func setCorsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}
