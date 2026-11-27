package auth

import (
	"context"
	"time"

	"github.com/voloshmiak/tamagochi/internal/platform/ratelimit"
)

const (
	MaxLoginFailures = 5
	LoginLockout     = 15 * time.Minute
	loginKeyPrefix   = "rl:login:"
)

type LoginGuard interface {
	Check(ctx context.Context, email string) error
	Failed(ctx context.Context, email string) error
	Succeeded(ctx context.Context, email string) error
}

type CounterGuard struct {
	Counter ratelimit.Counter
}

func (g CounterGuard) Check(ctx context.Context, email string) error {
	w, err := g.Counter.Get(ctx, loginKeyPrefix+email)
	if err != nil {
		return err
	}
	if w.Count >= MaxLoginFailures {
		return &ratelimit.LimitError{Wait: w.ResetIn}
	}
	return nil
}

func (g CounterGuard) Failed(ctx context.Context, email string) error {
	_, err := g.Counter.Incr(ctx, loginKeyPrefix+email, LoginLockout)
	return err
}

func (g CounterGuard) Succeeded(ctx context.Context, email string) error {
	return g.Counter.Reset(ctx, loginKeyPrefix+email)
}

type noGuard struct{}

func (noGuard) Check(context.Context, string) error     { return nil }
func (noGuard) Failed(context.Context, string) error    { return nil }
func (noGuard) Succeeded(context.Context, string) error { return nil }
