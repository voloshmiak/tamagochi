package auth

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/voloshmiak/tamagochi/internal/platform/clock"
)

var storeT0 = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func testRec(id, family string, ttl time.Duration) RefreshRecord {
	return RefreshRecord{ID: id, FamilyID: family, UserID: "u1", ExpiresAt: storeT0.Add(ttl)}
}

func TestRefreshStore_ConsumeOnce(t *testing.T) {
	s := NewMemoryRefreshStore(clock.NewManual(storeT0))
	ctx := context.Background()
	if err := s.Save(ctx, testRec("a", "f1", time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := s.Consume(ctx, "a")
	if err != nil || got.FamilyID != "f1" || got.UserID != "u1" {
		t.Fatalf("first consume: rec=%+v err=%v", got, err)
	}

	got, err = s.Consume(ctx, "a")
	if !errors.Is(err, ErrRefreshUsed) {
		t.Fatalf("second consume: want ErrRefreshUsed, got %v", err)
	}
	if got.FamilyID != "f1" {
		t.Fatalf("reuse must return the record, got %+v", got)
	}
}

func TestRefreshStore_UnknownAndExpired(t *testing.T) {
	clk := clock.NewManual(storeT0)
	s := NewMemoryRefreshStore(clk)
	ctx := context.Background()

	if _, err := s.Consume(ctx, "nope"); !errors.Is(err, ErrRefreshUnknown) {
		t.Fatalf("unknown token: got %v", err)
	}

	_ = s.Save(ctx, testRec("a", "f1", time.Hour))
	clk.Advance(time.Hour)
	if _, err := s.Consume(ctx, "a"); !errors.Is(err, ErrRefreshUnknown) {
		t.Fatalf("expired token: got %v", err)
	}
}

func TestRefreshStore_RevokeSessionKillsWholeFamily(t *testing.T) {
	s := NewMemoryRefreshStore(clock.NewManual(storeT0))
	ctx := context.Background()
	_ = s.Save(ctx, testRec("a", "f1", time.Hour))
	_, _ = s.Consume(ctx, "a")
	_ = s.Save(ctx, testRec("b", "f1", time.Hour))
	_ = s.Save(ctx, testRec("x", "f2", time.Hour))

	if err := s.RevokeSession(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Consume(ctx, "b"); !errors.Is(err, ErrRefreshUnknown) {
		t.Fatalf("token of revoked family must be rejected, got %v", err)
	}
	if err := s.Save(ctx, testRec("c", "f1", time.Hour)); !errors.Is(err, ErrRefreshRevoked) {
		t.Fatalf("saving into revoked family: want ErrRefreshRevoked, got %v", err)
	}
	if _, err := s.Consume(ctx, "x"); err != nil {
		t.Fatalf("other sessions must stay valid, got %v", err)
	}
}

func TestRefreshStore_RevokeUnknownIsNoop(t *testing.T) {
	s := NewMemoryRefreshStore(clock.NewManual(storeT0))
	if err := s.RevokeSession(context.Background(), "nope"); err != nil {
		t.Fatalf("revoking an unknown token must be a no-op, got %v", err)
	}
}

func TestRefreshStore_PrunesExpired(t *testing.T) {
	clk := clock.NewManual(storeT0)
	s := NewMemoryRefreshStore(clk)
	ctx := context.Background()
	_ = s.Save(ctx, testRec("a", "f1", time.Hour))
	_ = s.RevokeSession(ctx, "a")

	clk.Advance(2 * time.Hour)
	_ = s.Save(ctx, testRec("b", "f2", time.Hour))

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.tokens) != 1 || len(s.revoked) != 0 {
		t.Fatalf("expected 1 token and no revoked families, got %d and %d", len(s.tokens), len(s.revoked))
	}
}

func TestRefreshStore_ConcurrentConsumeHasOneWinner(t *testing.T) {
	s := NewMemoryRefreshStore(clock.NewManual(storeT0))
	ctx := context.Background()
	_ = s.Save(ctx, testRec("a", "f1", time.Hour))

	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Consume(ctx, "a"); err == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("exactly one concurrent consume must succeed, got %d", wins.Load())
	}
}
