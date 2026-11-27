package queue

import (
	"context"
	"time"
)

const MaxAttempts = 5

const (
	BaseDelay = 10 * time.Second
	MaxDelay  = 30 * time.Minute
)

const (
	TaskPing    = "system:ping"
	TaskReplica = "replica:generate"
	TaskLowStat = "pet:low-stat-check"
	TaskNotify  = "notify:send"
)

type Task struct {
	ID      string
	Type    string
	Payload []byte
	RunAt   time.Time
	Attempt int
}

type Enqueuer interface {
	Enqueue(ctx context.Context, t Task) error
}

type HandlerFunc func(ctx context.Context, t Task) error

type Registrar interface {
	Handle(taskType string, h HandlerFunc)
}

func Backoff(retried int) time.Duration {
	d := BaseDelay
	for range retried {
		d *= 2
		if d >= MaxDelay {
			return MaxDelay
		}
	}
	return d
}
