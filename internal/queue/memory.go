package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var ErrSkipRetry = errors.New("skip retry")

type Memory struct {
	metrics *Metrics
	delay   func(retried int) time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu       sync.Mutex
	handlers map[string]HandlerFunc
	pending  map[string]bool
	dead     []Task
}

func NewMemory(metrics *Metrics, delay func(retried int) time.Duration) *Memory {
	if metrics == nil {
		metrics = NewMetrics()
	}
	if delay == nil {
		delay = Backoff
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Memory{
		metrics:  metrics,
		delay:    delay,
		ctx:      ctx,
		cancel:   cancel,
		handlers: make(map[string]HandlerFunc),
		pending:  make(map[string]bool),
	}
}

func (q *Memory) Metrics() *Metrics { return q.metrics }

func (q *Memory) Handle(taskType string, h HandlerFunc) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[taskType] = q.metrics.Wrap(taskType, h)
}

func (q *Memory) Enqueue(_ context.Context, t Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.ctx.Err() != nil {
		return errors.New("queue is closed")
	}
	if t.ID != "" {
		if q.pending[t.ID] {
			return nil
		}
		q.pending[t.ID] = true
	}
	q.wg.Add(1)
	t.Attempt = 1
	q.schedule(t, time.Until(t.RunAt))
	return nil
}

func (q *Memory) schedule(t Task, wait time.Duration) {
	go func() {
		timer := time.NewTimer(max(wait, 0))
		defer timer.Stop()
		select {
		case <-q.ctx.Done():
			q.finish(t)
		case <-timer.C:
			q.run(t)
		}
	}()
}

func (q *Memory) run(t Task) {
	q.mu.Lock()
	h, ok := q.handlers[t.Type]
	q.mu.Unlock()
	err := fmt.Errorf("%w: no handler for task %q", ErrSkipRetry, t.Type)
	if ok {
		err = h(q.ctx, t)
	}
	if err == nil {
		q.finish(t)
		return
	}
	if t.Attempt >= MaxAttempts || errors.Is(err, ErrSkipRetry) {
		slog.Error("task moved to dead queue", "task", t.Type, "id", t.ID, "attempts", t.Attempt, "err", err)
		q.metrics.Observe(t.Type, OutcomeDead, 0)
		q.mu.Lock()
		q.dead = append(q.dead, t)
		q.mu.Unlock()
		q.finish(t)
		return
	}
	slog.Warn("task failed, will retry", "task", t.Type, "id", t.ID, "attempt", t.Attempt, "err", err)
	wait := q.delay(t.Attempt - 1)
	t.Attempt++
	q.schedule(t, wait)
}

func (q *Memory) finish(t Task) {
	if t.ID != "" {
		q.mu.Lock()
		delete(q.pending, t.ID)
		q.mu.Unlock()
	}
	q.wg.Done()
}

func (q *Memory) Dead() []Task {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]Task(nil), q.dead...)
}

func (q *Memory) Wait() { q.wg.Wait() }

func (q *Memory) Close() error {
	q.cancel()
	q.wg.Wait()
	return nil
}
