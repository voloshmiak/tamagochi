package queue

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"sync"
	"time"
)

type Outcome string

const (
	OutcomeSucceeded Outcome = "succeeded"
	OutcomeFailed    Outcome = "failed"
	OutcomeDead      Outcome = "dead"
)

type Metrics struct {
	mu       sync.Mutex
	counts   map[metricKey]int
	duration map[string]time.Duration
}

type metricKey struct {
	task    string
	outcome Outcome
}

func NewMetrics() *Metrics {
	return &Metrics{counts: make(map[metricKey]int), duration: make(map[string]time.Duration)}
}

func (m *Metrics) Observe(task string, outcome Outcome, took time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counts[metricKey{task, outcome}]++
	if outcome != OutcomeDead {
		m.duration[task] += took
	}
}

func (m *Metrics) Count(task string, outcome Outcome) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counts[metricKey{task, outcome}]
}

func (m *Metrics) Wrap(task string, h HandlerFunc) HandlerFunc {
	return func(ctx context.Context, t Task) error {
		start := time.Now()
		err := h(ctx, t)
		outcome := OutcomeSucceeded
		if err != nil {
			outcome = OutcomeFailed
		}
		m.Observe(task, outcome, time.Since(start))
		return err
	}
}

func (m *Metrics) WriteText(w io.Writer) error {
	m.mu.Lock()
	keys := slices.SortedFunc(maps.Keys(m.counts), func(a, b metricKey) int {
		if a.task != b.task {
			return cmp.Compare(a.task, b.task)
		}
		return cmp.Compare(a.outcome, b.outcome)
	})
	counts := maps.Clone(m.counts)
	durations := maps.Clone(m.duration)
	m.mu.Unlock()

	if _, err := fmt.Fprintln(w, "# TYPE pet_engine_tasks_total counter"); err != nil {
		return err
	}
	for _, k := range keys {
		if _, err := fmt.Fprintf(w, "pet_engine_tasks_total{task=%q,outcome=%q} %d\n", k.task, k.outcome, counts[k]); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "# TYPE pet_engine_task_seconds_total counter"); err != nil {
		return err
	}
	for _, task := range slices.Sorted(maps.Keys(durations)) {
		if _, err := fmt.Fprintf(w, "pet_engine_task_seconds_total{task=%q} %g\n", task, durations[task].Seconds()); err != nil {
			return err
		}
	}
	return nil
}

func (m *Metrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_ = m.WriteText(w)
}
