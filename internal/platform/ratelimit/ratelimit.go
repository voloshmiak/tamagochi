package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
)

type Window struct {
	Count   int
	ResetIn time.Duration
}

type Counter interface {
	Incr(ctx context.Context, key string, window time.Duration) (Window, error)
	Get(ctx context.Context, key string) (Window, error)
	Decr(ctx context.Context, key string) error
	Reset(ctx context.Context, key string) error
}

type LimitError struct {
	Wait time.Duration
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("too many requests; retry in %s", e.Wait.Round(time.Second))
}

func (e *LimitError) Unwrap() error { return apperr.ErrRateLimited }

func (e *LimitError) RetryAfter() time.Duration { return e.Wait }

type Memory struct {
	clock clock.Clock

	mu      sync.Mutex
	entries map[string]*memEntry
	calls   int
}

type memEntry struct {
	count   int
	resetAt time.Time
}

func NewMemory(c clock.Clock) *Memory {
	return &Memory{clock: c, entries: make(map[string]*memEntry)}
}

func (m *Memory) live(key string, now time.Time) *memEntry {
	e, ok := m.entries[key]
	if !ok {
		return nil
	}
	if !e.resetAt.After(now) {
		delete(m.entries, key)
		return nil
	}
	return e
}

func (m *Memory) prune(now time.Time) {
	m.calls++
	if m.calls%1024 != 0 {
		return
	}
	for key := range m.entries {
		m.live(key, now)
	}
}

func (m *Memory) Incr(_ context.Context, key string, window time.Duration) (Window, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.clock.Now()
	m.prune(now)
	e := m.live(key, now)
	if e == nil {
		e = &memEntry{resetAt: now.Add(window)}
		m.entries[key] = e
	}
	e.count++
	return Window{Count: e.count, ResetIn: e.resetAt.Sub(now)}, nil
}

func (m *Memory) Get(_ context.Context, key string) (Window, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.clock.Now()
	e := m.live(key, now)
	if e == nil {
		return Window{}, nil
	}
	return Window{Count: e.count, ResetIn: e.resetAt.Sub(now)}, nil
}

func (m *Memory) Decr(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.live(key, m.clock.Now()); e != nil && e.count > 0 {
		e.count--
	}
	return nil
}

func (m *Memory) Reset(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key)
	return nil
}

type Redis struct {
	Client redis.UniversalClient
}

var incrScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
local ttl = redis.call('PTTL', KEYS[1])
if ttl < 0 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
  ttl = tonumber(ARGV[1])
end
return {n, ttl}
`)

var decrScript = redis.NewScript(`
local n = tonumber(redis.call('GET', KEYS[1]) or '0')
if n > 0 then redis.call('DECR', KEYS[1]) end
return 1
`)

func (r Redis) Incr(ctx context.Context, key string, window time.Duration) (Window, error) {
	res, err := incrScript.Run(ctx, r.Client, []string{key}, window.Milliseconds()).Int64Slice()
	if err != nil {
		return Window{}, fmt.Errorf("rate counter %s: %w", key, err)
	}
	return Window{Count: int(res[0]), ResetIn: time.Duration(res[1]) * time.Millisecond}, nil
}

func (r Redis) Get(ctx context.Context, key string) (Window, error) {
	pipe := r.Client.Pipeline()
	get := pipe.Get(ctx, key)
	ttl := pipe.PTTL(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return Window{}, fmt.Errorf("rate counter %s: %w", key, err)
	}
	n, err := get.Int()
	if errors.Is(err, redis.Nil) {
		return Window{}, nil
	}
	if err != nil {
		return Window{}, fmt.Errorf("rate counter %s: %w", key, err)
	}
	return Window{Count: n, ResetIn: max(ttl.Val(), 0)}, nil
}

func (r Redis) Decr(ctx context.Context, key string) error {
	if err := decrScript.Run(ctx, r.Client, []string{key}).Err(); err != nil {
		return fmt.Errorf("rate counter %s: %w", key, err)
	}
	return nil
}

func (r Redis) Reset(ctx context.Context, key string) error {
	if err := r.Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("rate counter %s: %w", key, err)
	}
	return nil
}

type Limit struct {
	Max    int
	Window time.Duration
}

func Take(ctx context.Context, c Counter, key string, l Limit) (Window, error) {
	w, err := c.Incr(ctx, key, l.Window)
	if err != nil {
		return w, err
	}
	if w.Count > l.Max {
		return w, &LimitError{Wait: w.ResetIn}
	}
	return w, nil
}
