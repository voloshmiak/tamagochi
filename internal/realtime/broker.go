package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

const (
	channelPrefix = "rt:"
	bufferSize    = 16
)

type RedisBroker struct {
	Client redis.UniversalClient
}

func (b RedisBroker) Publish(ctx context.Context, topic string, msg []byte) error {
	if err := b.Client.Publish(ctx, channelPrefix+topic, msg).Err(); err != nil {
		return fmt.Errorf("publish %s: %w", topic, err)
	}
	return nil
}

func (b RedisBroker) Subscribe(ctx context.Context, topic string) (<-chan []byte, func()) {
	ctx, cancel := context.WithCancel(ctx)
	sub := b.Client.Subscribe(ctx, channelPrefix+topic)
	out := make(chan []byte, bufferSize)
	go func() {
		defer close(out)
		defer sub.Close()
		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-ch:
				if !ok {
					return
				}
				select {
				case out <- []byte(m.Payload):
				default:
				}
			}
		}
	}()
	return out, cancel
}

type MemoryBroker struct {
	mu   sync.Mutex
	subs map[string]map[chan []byte]struct{}
}

func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{subs: make(map[string]map[chan []byte]struct{})}
}

func (b *MemoryBroker) Publish(_ context.Context, topic string, msg []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[topic] {
		select {
		case ch <- msg:
		default:
		}
	}
	return nil
}

func (b *MemoryBroker) Subscribe(ctx context.Context, topic string) (<-chan []byte, func()) {
	ch := make(chan []byte, bufferSize)
	b.mu.Lock()
	if b.subs[topic] == nil {
		b.subs[topic] = make(map[chan []byte]struct{})
	}
	b.subs[topic][ch] = struct{}{}
	b.mu.Unlock()
	var once sync.Once
	done := make(chan struct{})
	stop := func() {
		once.Do(func() {
			close(done)
			b.mu.Lock()
			delete(b.subs[topic], ch)
			if len(b.subs[topic]) == 0 {
				delete(b.subs, topic)
			}
			b.mu.Unlock()
			close(ch)
		})
	}
	go func() {
		select {
		case <-ctx.Done():
			stop()
		case <-done:
		}
	}()
	return ch, stop
}

func PublishJSON(ctx context.Context, b Broker, topic string, m Message) {
	data, err := json.Marshal(m)
	if err == nil {
		err = b.Publish(ctx, topic, data)
	}
	if err != nil {
		slog.WarnContext(ctx, "publish realtime message", "topic", topic, "type", m.Type, "err", err)
	}
}
