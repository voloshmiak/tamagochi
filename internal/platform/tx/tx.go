package tx

import (
	"context"
	"sync"
)

type Manager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Local struct {
	mu sync.Mutex
}

func (l *Local) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return fn(ctx)
}
