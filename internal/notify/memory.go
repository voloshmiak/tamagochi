package notify

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"sync"
)

type MemorySubscriptions struct {
	mu         sync.RWMutex
	byEndpoint map[string]Subscription
}

func NewMemorySubscriptions() *MemorySubscriptions {
	return &MemorySubscriptions{byEndpoint: make(map[string]Subscription)}
}

func (m *MemorySubscriptions) Add(_ context.Context, s Subscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byEndpoint[s.Endpoint] = s
	return nil
}

func (m *MemorySubscriptions) Remove(_ context.Context, userID, endpoint string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.byEndpoint[endpoint]; ok && s.UserID == userID {
		delete(m.byEndpoint, endpoint)
	}
	return nil
}

func (m *MemorySubscriptions) RemoveEndpoint(_ context.Context, endpoint string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byEndpoint, endpoint)
	return nil
}

func (m *MemorySubscriptions) ByUser(_ context.Context, userID string) ([]Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Subscription
	for _, s := range m.byEndpoint {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b Subscription) int { return cmp.Compare(a.Endpoint, b.Endpoint) })
	return out, nil
}

type MemoryPreferences struct {
	mu     sync.RWMutex
	byUser map[string]Preferences
}

func NewMemoryPreferences() *MemoryPreferences {
	return &MemoryPreferences{byUser: make(map[string]Preferences)}
}

func (m *MemoryPreferences) Get(_ context.Context, userID string) (Preferences, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.byUser[userID]; ok {
		return p.Clone(), nil
	}
	return DefaultPreferences(), nil
}

func (m *MemoryPreferences) Save(_ context.Context, userID string, p Preferences) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p.Disabled = maps.Clone(p.Disabled)
	m.byUser[userID] = p
	return nil
}
