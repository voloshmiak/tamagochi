package tx

import (
	"context"
	"sync"
)

type Manager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type hooksKey struct{}

type Hooks struct {
	mu  sync.Mutex
	fns []func(context.Context)
}

func WithHooks(ctx context.Context) (context.Context, *Hooks) {
	h := &Hooks{}
	return context.WithValue(ctx, hooksKey{}, h), h
}

func InTx(ctx context.Context) bool {
	_, ok := ctx.Value(hooksKey{}).(*Hooks)
	return ok
}

func AfterCommit(ctx context.Context, fn func(ctx context.Context)) {
	if h, ok := ctx.Value(hooksKey{}).(*Hooks); ok {
		h.mu.Lock()
		h.fns = append(h.fns, fn)
		h.mu.Unlock()
		return
	}
	fn(ctx)
}

func (h *Hooks) Run(ctx context.Context) {
	h.mu.Lock()
	fns := h.fns
	h.fns = nil
	h.mu.Unlock()
	ctx = context.WithoutCancel(ctx)
	for _, fn := range fns {
		fn(ctx)
	}
}

type Local struct {
	mu sync.Mutex
}

func (l *Local) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if InTx(ctx) {
		return fn(ctx)
	}
	txCtx, hooks := WithHooks(ctx)
	if err := l.run(txCtx, fn); err != nil {
		return err
	}
	hooks.Run(ctx)
	return nil
}

func (l *Local) run(ctx context.Context, fn func(ctx context.Context) error) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return fn(ctx)
}
