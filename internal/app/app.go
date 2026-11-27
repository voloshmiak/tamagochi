// Package app is the composition root: it picks the adapters (PostgreSQL and
// Redis, or in-memory ones for a quick local run), wires the modules and
// builds the HTTP router from the OpenAPI contract.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/voloshmiak/tamagochi/internal/api/oapi"
	"github.com/voloshmiak/tamagochi/internal/auth"
	"github.com/voloshmiak/tamagochi/internal/brain"
	"github.com/voloshmiak/tamagochi/internal/config"
	"github.com/voloshmiak/tamagochi/internal/pets"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/httpx"
	"github.com/voloshmiak/tamagochi/internal/platform/pg"
	"github.com/voloshmiak/tamagochi/internal/platform/tx"
)

// BasePath is the prefix of every API route (servers[0].url in the spec).
const BasePath = "/api/v1"

// App is the wired application.
type App struct {
	Handler http.Handler
	closers []func() error
}

// Close releases the database and Redis connections.
func (a *App) Close() error {
	var errs []error
	for _, c := range a.closers {
		errs = append(errs, c())
	}
	return errors.Join(errs...)
}

// New wires the application. With cfg.DatabaseURL set it connects to
// PostgreSQL and, unless disabled, applies pending migrations; with
// cfg.RedisURL set refresh sessions are kept in Redis.
func New(ctx context.Context, cfg config.Config, c clock.Clock) (*App, error) {
	a := &App{}
	st, err := a.storage(ctx, cfg, c)
	if err != nil {
		_ = a.Close()
		return nil, err
	}

	tokens := &auth.JWTIssuer{Secret: []byte(cfg.JWTSecret), AccessTTL: cfg.AccessTTL, RefreshTTL: cfg.RefreshTTL, Clock: c}
	var mailer auth.Mailer = auth.LogMailer{}
	if cfg.SMTP.Addr != "" {
		mailer = auth.SMTPMailer{Addr: cfg.SMTP.Addr, From: cfg.SMTP.From, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password}
	}
	authSvc := auth.NewService(auth.Deps{
		Users:         st.users,
		Verifications: st.verifications,
		Hasher:        auth.BcryptHasher{},
		Tokens:        tokens,
		Mailer:        mailer,
		Refresh:       st.refresh,
		Clock:         c,
		VerifyURL:     cfg.PublicURL + BasePath + "/auth/verify",
	})
	petSvc := pets.NewService(pets.Deps{
		Pets:      st.pets,
		Inventory: st.inventory,
		Tx:        st.tx,
		Brain:     brain.NewUtilityBrain(cfg.BrainSeed),
		Clock:     c,
	})

	h, err := Router(tokens, Server{
		Handler:     auth.NewHandler(authSvc, cfg.CookieSecure, cfg.PublicURL),
		PetsHandler: pets.NewHandler(petSvc, c),
	})
	if err != nil {
		_ = a.Close()
		return nil, err
	}
	a.Handler = h
	return a, nil
}

type storage struct {
	users         auth.UserRepository
	verifications auth.VerificationRepository
	refresh       auth.RefreshStore
	pets          pets.PetRepository
	inventory     pets.InventoryRepository
	tx            tx.Manager
}

func (a *App) storage(ctx context.Context, cfg config.Config, c clock.Clock) (storage, error) {
	var st storage
	if cfg.DatabaseURL == "" {
		slog.WarnContext(ctx, "DATABASE_URL is not set: data is kept in memory and lost on restart")
		st.users = auth.NewMemoryUserRepository()
		st.verifications = auth.NewMemoryVerificationRepository()
		st.pets = pets.NewMemoryPetRepository()
		st.inventory = pets.NewMemoryInventoryRepository()
		st.tx = &tx.Local{}
	} else {
		pool, err := pg.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return st, err
		}
		a.closers = append(a.closers, func() error { pool.Close(); return nil })
		if cfg.MigrateOnStart {
			if err := pg.Migrate(ctx, pool); err != nil {
				return st, err
			}
		}
		st = postgresStorage(pool)
	}

	if cfg.RedisURL == "" {
		slog.WarnContext(ctx, "REDIS_URL is not set: refresh sessions are kept in memory")
		st.refresh = auth.NewMemoryRefreshStore(c)
		return st, nil
	}
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return st, fmt.Errorf("REDIS_URL: %w", err)
	}
	client := redis.NewClient(opts)
	a.closers = append(a.closers, client.Close)
	if err := client.Ping(ctx).Err(); err != nil {
		return st, fmt.Errorf("redis ping: %w", err)
	}
	st.refresh = auth.RedisRefreshStore{Client: client, Clock: c}
	return st, nil
}

func postgresStorage(pool *pgxpool.Pool) storage {
	return storage{
		users:         auth.PgUserRepository{Pool: pool},
		verifications: auth.PgVerificationRepository{Pool: pool},
		pets:          pets.PgPetRepository{Pool: pool},
		inventory:     pets.PgInventoryRepository{Pool: pool},
		tx:            pg.TxManager{Pool: pool},
	}
}

// Server implements oapi.ServerInterface by combining the module handlers.
// A new operation in the spec fails to compile until a module implements it.
type Server struct {
	*auth.Handler
	*PetsHandler
}

// PetsHandler is an alias that lets Server embed both module handlers,
// which share the type name Handler.
type PetsHandler = pets.Handler

var _ oapi.ServerInterface = Server{}

// Router builds the HTTP handler: routes come from the generated
// oapi.ServerInterface; every request is first authenticated (unless the spec
// marks the operation `security: []`) and then validated against the spec.
func Router(tokens httpx.TokenVerifier, srv oapi.ServerInterface) (http.Handler, error) {
	spec, err := oapi.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI spec: %w", err)
	}
	public := httpx.PublicOperations(spec, BasePath)
	isPublic := func(r *http.Request) bool {
		return public[r.Method+" "+chi.RouteContext(r.Context()).RoutePattern()]
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	notFound := func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusNotFound, httpx.ErrorBody{Error: "not_found", Message: "no such endpoint"})
	}
	r.NotFound(notFound)
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusMethodNotAllowed, httpx.ErrorBody{Error: "method_not_allowed", Message: r.Method + " is not allowed here"})
	})

	return oapi.HandlerWithOptions(srv, oapi.ChiServerOptions{
		BaseURL:    BasePath,
		BaseRouter: r,
		// The last middleware runs first: authenticate, then validate.
		Middlewares: []oapi.MiddlewareFunc{
			httpx.ValidateRequests(spec, BasePath),
			httpx.Authenticate(tokens, isPublic),
		},
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			httpx.JSON(w, http.StatusBadRequest, httpx.ErrorBody{Error: "invalid", Message: err.Error()})
		},
	}), nil
}
