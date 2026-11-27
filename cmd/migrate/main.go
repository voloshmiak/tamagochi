package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/voloshmiak/tamagochi/internal/platform/pg"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(context.Background()); err != nil {
		slog.Error("migrate failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL is required")
	}
	pool, err := pg.Open(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()
	return pg.Migrate(ctx, pool)
}
