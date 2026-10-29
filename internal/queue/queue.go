package queue

import (
	"context"
	"time"
)

const MaxAttempts = 5

const (
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
