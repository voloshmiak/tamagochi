package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/voloshmiak/tamagochi/internal/app"
	"github.com/voloshmiak/tamagochi/internal/config"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
)

const usage = `usage: pet-engine [command]

commands:
  (none)  run the background worker
  ping    enqueue a test task and exit
  vapid   print a new VAPID key pair for web push`

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(os.Args[1:]); err != nil {
		slog.Error("pet-engine stopped", "err", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}
	if cmd == "vapid" {
		priv, pub, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			return err
		}
		fmt.Printf("VAPID_PUBLIC_KEY=%s\nVAPID_PRIVATE_KEY=%s\n", pub, priv)
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch cmd {
	case "":
	case "ping":
		if err := app.EnqueuePing(ctx, cfg, clock.System{}); err != nil {
			return err
		}
		slog.Info("ping task enqueued")
		return nil
	default:
		fmt.Fprintln(os.Stderr, usage)
		return fmt.Errorf("unknown command %q", cmd)
	}

	w, err := app.NewWorker(ctx, cfg, clock.System{})
	if err != nil {
		return err
	}
	defer func() {
		if err := w.Close(); err != nil {
			slog.Error("close worker", "err", err)
		}
	}()
	slog.Info("pet-engine started", "concurrency", cfg.WorkerConcurrency)
	return w.Run(ctx)
}
