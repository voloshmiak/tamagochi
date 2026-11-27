package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
)

func queueOf(taskType string) string {
	if taskType == TaskNotify {
		return QueueCritical
	}
	return QueueDefault
}

type AsynqEnqueuer struct {
	client *asynq.Client
}

func NewAsynqEnqueuer(rdb redis.UniversalClient) *AsynqEnqueuer {
	return &AsynqEnqueuer{client: asynq.NewClientFromRedisClient(rdb)}
}

func (e *AsynqEnqueuer) Enqueue(ctx context.Context, t Task) error {
	opts := []asynq.Option{asynq.MaxRetry(MaxAttempts - 1), asynq.Queue(queueOf(t.Type))}
	if t.ID != "" {
		opts = append(opts, asynq.TaskID(t.ID))
	}
	if !t.RunAt.IsZero() {
		opts = append(opts, asynq.ProcessAt(t.RunAt))
	}
	_, err := e.client.EnqueueContext(ctx, asynq.NewTask(t.Type, t.Payload), opts...)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", t.Type, err)
	}
	return nil
}

func (e *AsynqEnqueuer) Close() error { return e.client.Close() }

type Worker struct {
	server  *asynq.Server
	mux     *asynq.ServeMux
	metrics *Metrics
}

func NewWorker(rdb redis.UniversalClient, concurrency int, metrics *Metrics) *Worker {
	w := &Worker{mux: asynq.NewServeMux(), metrics: metrics}
	w.server = asynq.NewServerFromRedisClient(rdb, asynq.Config{
		Concurrency:    concurrency,
		Queues:         map[string]int{QueueCritical: 6, QueueDefault: 3},
		RetryDelayFunc: func(n int, _ error, _ *asynq.Task) time.Duration { return Backoff(n) },
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			retried, _ := asynq.GetRetryCount(ctx)
			maxRetry, _ := asynq.GetMaxRetry(ctx)
			id, _ := asynq.GetTaskID(ctx)
			if retried >= maxRetry || errors.Is(err, asynq.SkipRetry) {
				metrics.Observe(task.Type(), OutcomeDead, 0)
				slog.ErrorContext(ctx, "task moved to dead queue", "task", task.Type(), "id", id, "attempts", retried+1, "err", err)
				return
			}
			slog.WarnContext(ctx, "task failed, will retry", "task", task.Type(), "id", id, "attempt", retried+1, "err", err)
		}),
		Logger: slogAdapter{},
	})
	return w
}

func (w *Worker) Handle(taskType string, h HandlerFunc) {
	wrapped := w.metrics.Wrap(taskType, h)
	w.mux.HandleFunc(taskType, func(ctx context.Context, task *asynq.Task) error {
		retried, _ := asynq.GetRetryCount(ctx)
		id, _ := asynq.GetTaskID(ctx)
		err := wrapped(ctx, Task{ID: id, Type: task.Type(), Payload: task.Payload(), Attempt: retried + 1})
		if errors.Is(err, ErrSkipRetry) {
			return fmt.Errorf("%v: %w", err, asynq.SkipRetry)
		}
		return err
	})
}

func (w *Worker) Run(ctx context.Context) error {
	if err := w.server.Start(w.mux); err != nil {
		return fmt.Errorf("start worker: %w", err)
	}
	<-ctx.Done()
	w.server.Shutdown()
	return nil
}

func (w *Worker) Ping() error { return w.server.Ping() }

type slogAdapter struct{}

func (slogAdapter) Debug(args ...any) { slog.Debug(fmt.Sprint(args...)) }
func (slogAdapter) Info(args ...any)  { slog.Info(fmt.Sprint(args...)) }
func (slogAdapter) Warn(args ...any)  { slog.Warn(fmt.Sprint(args...)) }
func (slogAdapter) Error(args ...any) { slog.Error(fmt.Sprint(args...)) }
func (slogAdapter) Fatal(args ...any) { slog.Error(fmt.Sprint(args...)) }
