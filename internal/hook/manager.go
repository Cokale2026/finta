package hook

import (
	"context"
	"sort"
	"sync"
)

// Manager manages hook handlers and triggers
type Manager struct {
	handlers map[HookPoint][]Handler
	mu       sync.RWMutex
}

// NewManager creates a new hook manager
func NewManager() *Manager {
	return &Manager{
		handlers: make(map[HookPoint][]Handler),
	}
}

// Register adds a handler to the manager
func (m *Manager) Register(handler Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, point := range handler.Points() {
		m.handlers[point] = append(m.handlers[point], handler)
	}

	// Sort by priority (higher first)
	for point := range m.handlers {
		sort.Slice(m.handlers[point], func(i, j int) bool {
			return m.handlers[point][i].Priority() > m.handlers[point][j].Priority()
		})
	}
}

// Trigger executes all handlers for a hook point
// Returns the combined feedback - if any handler denies, the result denies
func (m *Manager) Trigger(ctx context.Context, data *HookData) (*Feedback, error) {
	m.mu.RLock()
	handlers := m.handlers[data.Point]
	m.mu.RUnlock()

	if len(handlers) == 0 {
		return AllowFeedback(), nil
	}

	// Execute handlers in priority order
	for _, handler := range handlers {
		feedback, err := handler.Handle(ctx, data)
		if err != nil {
			return nil, err
		}

		// If handler denies, stop and return
		if !feedback.Allow {
			return feedback, nil
		}

		// If handler modified data, update for next handler
		if feedback.Modified != nil {
			data.Data["_modified"] = feedback.Modified
		}
	}

	return AllowFeedback(), nil
}

// HasHandlers checks if there are handlers for a hook point
func (m *Manager) HasHandlers(point HookPoint) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.handlers[point]) > 0
}

// ListHandlers returns handler names for a hook point
func (m *Manager) ListHandlers(point HookPoint) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	handlers := m.handlers[point]
	names := make([]string, len(handlers))
	for i, h := range handlers {
		names[i] = h.Name()
	}
	return names
}
