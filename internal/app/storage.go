package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/voloshmiak/tamagochi/internal/auth"
	"github.com/voloshmiak/tamagochi/internal/config"
	"github.com/voloshmiak/tamagochi/internal/notify"
	"github.com/voloshmiak/tamagochi/internal/pets"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/pg"
	"github.com/voloshmiak/tamagochi/internal/platform/ratelimit"
	"github.com/voloshmiak/tamagochi/internal/platform/tx"
	"github.com/voloshmiak/tamagochi/internal/realtime"
)

type storage struct {
	users         auth.UserRepository
	verifications auth.VerificationRepository
	identities    auth.IdentityRepository
	refresh       auth.RefreshStore
	pets          pets.PetRepository
	inventory     pets.InventoryRepository
	subscriptions notify.SubscriptionStore
	preferences   notify.PreferenceStore
	counter       ratelimit.Counter
	broker        realtime.Broker
	tx            tx.Manager
	redis         redis.UniversalClient
}

type closerSink interface {
	addCloser(func() error)
}

func (a *App) addCloser(f func() error) { a.closers = append(a.closers, f) }

func openStorage(ctx context.Context, cfg config.Config, c clock.Clock, sink closerSink) (storage, error) {
	var st storage
	if cfg.DatabaseURL == "" {
		slog.WarnContext(ctx, "DATABASE_URL is not set: data is kept in memory and lost on restart")
		st.users = auth.NewMemoryUserRepository()
		st.verifications = auth.NewMemoryVerificationRepository()
		st.identities = auth.NewMemoryIdentityRepository()
		st.pets = pets.NewMemoryPetRepository()
		st.inventory = pets.NewMemoryInventoryRepository()
		st.subscriptions = notify.NewMemorySubscriptions()
		st.preferences = notify.NewMemoryPreferences()
		st.tx = &tx.Local{}
	} else {
		pool, err := pg.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return st, err
		}
		sink.addCloser(func() error { pool.Close(); return nil })
		if cfg.MigrateOnStart {
			if err := pg.Migrate(ctx, pool); err != nil {
				return st, err
			}
		}
		st = postgresStorage(pool)
	}

	if cfg.RedisURL == "" {
		slog.WarnContext(ctx, "REDIS_URL is not set: sessions, rate limits and pub/sub live in this process only")
		st.refresh = auth.NewMemoryRefreshStore(c)
		st.counter = ratelimit.NewMemory(c)
		st.broker = realtime.NewMemoryBroker()
		return st, nil
	}
	client, err := OpenRedis(ctx, cfg.RedisURL)
	if err != nil {
		return st, err
	}
	sink.addCloser(client.Close)
	st.redis = client
	st.refresh = auth.RedisRefreshStore{Client: client, Clock: c}
	st.counter = ratelimit.Redis{Client: client}
	st.broker = realtime.RedisBroker{Client: client}
	return st, nil
}

func OpenRedis(ctx context.Context, url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("REDIS_URL: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}

func postgresStorage(pool *pgxpool.Pool) storage {
	return storage{
		users:         auth.PgUserRepository{Pool: pool},
		verifications: auth.PgVerificationRepository{Pool: pool},
		identities:    auth.PgIdentityRepository{Pool: pool},
		pets:          pets.PgPetRepository{Pool: pool},
		inventory:     pets.PgInventoryRepository{Pool: pool},
		subscriptions: notify.PgSubscriptions{Pool: pool},
		preferences:   notify.PgPreferences{Pool: pool},
		tx:            pg.TxManager{Pool: pool},
	}
}
