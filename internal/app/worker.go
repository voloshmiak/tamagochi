package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/voloshmiak/tamagochi/internal/config"
	"github.com/voloshmiak/tamagochi/internal/engine"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/queue"
)

type Worker struct {
	worker  *queue.Worker
	metrics *queue.Metrics
	addr    string
	closers []func() error
}

func (w *Worker) addCloser(f func() error) { w.closers = append(w.closers, f) }

func (w *Worker) Close() error {
	var errs []error
	for i := len(w.closers) - 1; i >= 0; i-- {
		errs = append(errs, w.closers[i]())
	}
	return errors.Join(errs...)
}

func NewWorker(ctx context.Context, cfg config.Config, c clock.Clock) (*Worker, error) {
	if cfg.DatabaseURL == "" || cfg.RedisURL == "" {
		return nil, errors.New("pet-engine needs DATABASE_URL and REDIS_URL")
	}
	w := &Worker{metrics: queue.NewMetrics(), addr: cfg.MetricsAddr}
	st, err := openStorage(ctx, cfg, c, w)
	if err != nil {
		_ = w.Close()
		return nil, err
	}
	enq := queue.NewAsynqEnqueuer(st.redis)
	w.addCloser(enq.Close)
	w.worker = queue.NewWorker(st.redis, cfg.WorkerConcurrency, w.metrics)
	engine.New(engine.Deps{
		Pets:   st.pets,
		Notify: newNotifyService(cfg, st, c),
		Queue:  enq,
		Users:  engine.UserNicknames{Users: st.users},
		Clock:  c,
	}).Register(w.worker)
	return w, nil
}

func (w *Worker) Metrics() *queue.Metrics { return w.metrics }

func (w *Worker) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", w.metrics)
	mux.HandleFunc("GET /healthz", func(rw http.ResponseWriter, _ *http.Request) {
		if err := w.worker.Ping(); err != nil {
			http.Error(rw, err.Error(), http.StatusServiceUnavailable)
			return
		}
		rw.WriteHeader(http.StatusOK)
	})
	srv := &http.Server{Addr: w.addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	errCh := make(chan error, 2)
	go func() {
		slog.Info("pet-engine metrics listening", "addr", w.addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("metrics server: %w", err)
		}
	}()
	go func() { errCh <- w.worker.Run(ctx) }()

	var err error
	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = <-errCh
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return errors.Join(err, srv.Shutdown(shutdownCtx))
}

func EnqueuePing(ctx context.Context, cfg config.Config, c clock.Clock) error {
	if cfg.RedisURL == "" {
		return errors.New("REDIS_URL is required to enqueue a task")
	}
	client, err := OpenRedis(ctx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer client.Close()
	enq := queue.NewAsynqEnqueuer(client)
	defer enq.Close()
	return enq.Enqueue(ctx, engine.PingTask(c.Now()))
}
