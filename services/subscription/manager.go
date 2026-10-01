package subscription

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// Item holds subscription metadata for JSON API output.
type Item struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Status  string `json:"status"`
	sub     *nats.Subscription
}

// Manager manages in-memory lifecycle of NATS subscriptions.
type Manager struct {
	nc    *nats.Conn
	mu    sync.RWMutex
	items map[string]*Item
	seqID uint64
}

// NewManager creates a new Subscription Manager instance.
func NewManager(nc *nats.Conn) *Manager {
	return &Manager{
		nc:    nc,
		items: make(map[string]*Item),
	}
}

// Subscribe starts a new NATS subscription for the given subject.
func (m *Manager) Subscribe(sub *nats.Subscription) (*Item, error) {
	id := fmt.Sprintf("sub-%d", atomic.AddUint64(&m.seqID, 1))
	item := &Item{
		ID:      id,
		Subject: sub.Subject,
		Status:  "ACTIVE",
		sub:     sub,
	}

	m.mu.Lock()
	m.items[id] = item
	m.mu.Unlock()

	log.Info().Str("id", id).Str("subject", sub.Subject).Msg("Subscription registered")
	return &Item{ID: item.ID, Subject: item.Subject, Status: item.Status}, nil
}

// List returns all registered subscriptions.
func (m *Manager) List() []*Item {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Item, 0, len(m.items))
	for _, item := range m.items {
		result = append(result, &Item{
			ID:      item.ID,
			Subject: item.Subject,
			Status:  item.Status,
		})
	}
	return result
}

// Unsubscribe immediately stops a subscription by ID.
func (m *Manager) Unsubscribe(id string) error {
	m.mu.Lock()
	item, ok := m.items[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("subscription ID %s not found", id)
	}
	delete(m.items, id)
	m.mu.Unlock()

	if err := item.sub.Unsubscribe(); err != nil {
		return fmt.Errorf("failed to unsubscribe %s: %w", id, err)
	}

	log.Info().Str("id", id).Str("subject", item.Subject).Msg("Subscription unsubscribed")
	return nil
}

// Drain gracefully drains a subscription by ID allowing in-flight tasks to finish.
func (m *Manager) Drain(id string) error {
	m.mu.Lock()
	item, ok := m.items[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("subscription ID %s not found", id)
	}
	item.Status = "DRAINING"
	m.mu.Unlock()

	if err := item.sub.Drain(); err != nil {
		return fmt.Errorf("failed to drain subscription %s: %w", id, err)
	}

	log.Info().Str("id", id).Str("subject", item.Subject).Msg("Subscription drain initiated")
	return nil
}

// DrainAll drains all active subscriptions (used during server shutdown).
func (m *Manager) DrainAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, item := range m.items {
		_ = item.sub.Drain()
	}
}
