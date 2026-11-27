package auth

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/voloshmiak/tamagochi/internal/platform/clock"
)

var (
	ErrRefreshUnknown = errors.New("refresh token unknown or expired")
	ErrRefreshUsed    = errors.New("refresh token already used")
	ErrRefreshRevoked = errors.New("refresh session revoked")
)

type RefreshRecord struct {
	ID        string
	FamilyID  string
	UserID    string
	ExpiresAt time.Time
}

type RefreshStore interface {
	Save(ctx context.Context, rec RefreshRecord) error
	Consume(ctx context.Context, id string) (RefreshRecord, error)
	RevokeSession(ctx context.Context, id string) error
}

type MemoryRefreshStore struct {
	clock clock.Clock

	mu      sync.Mutex
	tokens  map[string]*refreshEntry
	revoked map[string]time.Time
}

type refreshEntry struct {
	rec  RefreshRecord
	used bool
}

func NewMemoryRefreshStore(c clock.Clock) *MemoryRefreshStore {
	return &MemoryRefreshStore{
		clock:   c,
		tokens:  make(map[string]*refreshEntry),
		revoked: make(map[string]time.Time),
	}
}

func (s *MemoryRefreshStore) Save(_ context.Context, rec RefreshRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune()
	if _, ok := s.revoked[rec.FamilyID]; ok {
		return ErrRefreshRevoked
	}
	s.tokens[rec.ID] = &refreshEntry{rec: rec}
	return nil
}

func (s *MemoryRefreshStore) Consume(_ context.Context, id string) (RefreshRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.tokens[id]
	if !ok {
		return RefreshRecord{}, ErrRefreshUnknown
	}
	if !e.rec.ExpiresAt.After(s.clock.Now()) {
		delete(s.tokens, id)
		return RefreshRecord{}, ErrRefreshUnknown
	}
	if e.used {
		return e.rec, ErrRefreshUsed
	}
	e.used = true
	return e.rec, nil
}

func (s *MemoryRefreshStore) RevokeSession(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.tokens[id]
	if !ok {
		return nil
	}
	family, until := e.rec.FamilyID, e.rec.ExpiresAt
	for tokenID, other := range s.tokens {
		if other.rec.FamilyID != family {
			continue
		}
		if other.rec.ExpiresAt.After(until) {
			until = other.rec.ExpiresAt
		}
		delete(s.tokens, tokenID)
	}
	s.revoked[family] = until
	return nil
}

func (s *MemoryRefreshStore) prune() {
	now := s.clock.Now()
	for id, e := range s.tokens {
		if !e.rec.ExpiresAt.After(now) {
			delete(s.tokens, id)
		}
	}
	for family, until := range s.revoked {
		if !until.After(now) {
			delete(s.revoked, family)
		}
	}
}
