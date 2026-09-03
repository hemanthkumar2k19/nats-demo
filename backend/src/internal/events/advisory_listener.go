package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// AdvisoryEvent represents a normalized NATS operational event or JetStream advisory.
type AdvisoryEvent struct {
	Timestamp string          `json:"timestamp"`
	Subject   string          `json:"subject"`
	EventType string          `json:"event_type"`
	Stream    string          `json:"stream,omitempty"`
	Consumer  string          `json:"consumer,omitempty"`
	Server    string          `json:"server,omitempty"`
	Account   string          `json:"account,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// AdvisoryListener listens to NATS $SYS and JetStream advisory subjects and pushes normalized records to Loki.
type AdvisoryListener struct {
	nc         *nats.Conn
	lokiURL    string
	httpClient *http.Client
	subs       []*nats.Subscription
	mu         sync.Mutex
	closed     bool
}

// NewAdvisoryListener initializes an operational event listener.
func NewAdvisoryListener(nc *nats.Conn) *AdvisoryListener {
	lokiURL := os.Getenv("LOKI_URL")
	if lokiURL == "" {
		lokiURL = "http://localhost:3100"
	}
	lokiURL = strings.TrimRight(lokiURL, "/")

	return &AdvisoryListener{
		nc:      nc,
		lokiURL: lokiURL,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
		subs: make([]*nats.Subscription, 0),
	}
}

// Start begins listening to $SYS and JetStream advisory event subjects.
func (l *AdvisoryListener) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	subjects := []string{
		"$JS.EVENT.ADVISORY.>",
		"$SYS.ACCOUNT.*.CONNECT",
		"$SYS.ACCOUNT.*.DISCONNECT",
	}

	for _, subj := range subjects {
		sub, err := l.nc.Subscribe(subj, l.handleEvent)
		if err != nil {
			log.Printf("[AdvisoryListener] Warning: failed to subscribe to %s: %v", subj, err)
			continue
		}
		l.subs = append(l.subs, sub)
		log.Printf("[AdvisoryListener] Subscribed to operational event subject: %s", subj)
	}

	return nil
}

func (l *AdvisoryListener) handleEvent(msg *nats.Msg) {
	evt := l.normalizeEvent(msg.Subject, msg.Data)

	// Asynchronously push to Loki
	go l.pushToLoki(evt)
}

func (l *AdvisoryListener) normalizeEvent(subject string, data []byte) AdvisoryEvent {
	now := time.Now().UTC()
	evt := AdvisoryEvent{
		Timestamp: now.Format(time.RFC3339Nano),
		Subject:   subject,
		EventType: "UNKNOWN",
	}

	if len(data) > 0 && json.Valid(data) {
		evt.Payload = json.RawMessage(data)
	}

	tokens := strings.Split(subject, ".")

	if strings.HasPrefix(subject, "$JS.EVENT.ADVISORY.") {
		// Example: $JS.EVENT.ADVISORY.CONSUMER.CREATED.JOBS.job-processor
		// Tokens: [0:$JS, 1:EVENT, 2:ADVISORY, 3:RESOURCE_TYPE, 4:ACTION, 5:STREAM, 6:CONSUMER]
		if len(tokens) >= 5 {
			evt.EventType = fmt.Sprintf("%s_%s", tokens[3], tokens[4])
		} else if len(tokens) >= 4 {
			evt.EventType = tokens[3]
		}
		if len(tokens) >= 6 {
			evt.Stream = tokens[5]
		}
		if len(tokens) >= 7 {
			evt.Consumer = tokens[6]
		}
	} else if strings.HasPrefix(subject, "$SYS.ACCOUNT.") {
		// Example: $SYS.ACCOUNT.DEFAULT.CONNECT
		if len(tokens) >= 4 {
			evt.Account = tokens[2]
			evt.EventType = fmt.Sprintf("CLIENT_%s", tokens[3])
		}
	}

	// Try extracting server from payload if available
	if len(data) > 0 {
		var meta struct {
			Server string `json:"server,omitempty"`
			Host   string `json:"host,omitempty"`
			ID     string `json:"id,omitempty"`
		}
		if err := json.Unmarshal(data, &meta); err == nil {
			if meta.Server != "" {
				evt.Server = meta.Server
			} else if meta.Host != "" {
				evt.Server = meta.Host
			}
		}
	}

	return evt
}

// pushToLoki formats the normalized advisory event for Loki's push API.
func (l *AdvisoryListener) pushToLoki(evt AdvisoryEvent) {
	evtBytes, err := json.Marshal(evt)
	if err != nil {
		return
	}

	nowNano := strconv.FormatInt(time.Now().UnixNano(), 10)

	labels := map[string]string{
		"service":      "nats-events",
		"event_source": "nats",
		"event_type":   evt.EventType,
		"environment":  "demo",
	}
	if evt.Stream != "" {
		labels["stream"] = evt.Stream
	}
	if evt.Consumer != "" {
		labels["consumer"] = evt.Consumer
	}

	pushReq := map[string]interface{}{
		"streams": []map[string]interface{}{
			{
				"stream": labels,
				"values": [][]string{
					{nowNano, string(evtBytes)},
				},
			},
		},
	}

	body, err := json.Marshal(pushReq)
	if err != nil {
		return
	}

	url := fmt.Sprintf("%s/loki/api/v1/push", l.lokiURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		// Quietly ignore network failures if Loki is not running in local demo
		return
	}
	defer resp.Body.Close()
}

// Close unsubscribes all active advisory listeners.
func (l *AdvisoryListener) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return
	}
	l.closed = true

	for _, sub := range l.subs {
		_ = sub.Unsubscribe()
	}
	l.subs = nil
	log.Println("[AdvisoryListener] Closed all operational event subscriptions")
}
