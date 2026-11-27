package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/voloshmiak/tamagochi/internal/api/oapi"
	"github.com/voloshmiak/tamagochi/internal/auth"
	"github.com/voloshmiak/tamagochi/internal/brain"
	"github.com/voloshmiak/tamagochi/internal/config"
	"github.com/voloshmiak/tamagochi/internal/engine"
	"github.com/voloshmiak/tamagochi/internal/notify"
	"github.com/voloshmiak/tamagochi/internal/pets"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/httpx"
	"github.com/voloshmiak/tamagochi/internal/platform/ratelimit"
	"github.com/voloshmiak/tamagochi/internal/queue"
	"github.com/voloshmiak/tamagochi/internal/realtime"
	"github.com/voloshmiak/tamagochi/internal/trade"
)

const BasePath = "/api/v1"

var _ trade.InventoryPort = (*pets.Service)(nil)

type App struct {
	Handler http.Handler
	closers []func() error
}

func (a *App) Close() error {
	var errs []error
	for i := len(a.closers) - 1; i >= 0; i-- {
		errs = append(errs, a.closers[i]())
	}
	return errors.Join(errs...)
}

func New(ctx context.Context, cfg config.Config, c clock.Clock) (*App, error) {
	a := &App{}
	h, err := a.build(ctx, cfg, c)
	if err != nil {
		_ = a.Close()
		return nil, err
	}
	a.Handler = h
	return a, nil
}

func (a *App) build(ctx context.Context, cfg config.Config, c clock.Clock) (http.Handler, error) {
	st, err := openStorage(ctx, cfg, c, a)
	if err != nil {
		return nil, err
	}
	q := a.queue(st)
	notifier := engine.Notifier{Queue: q}
	if st.redis == nil && cfg.VAPID.PublicKey == "" {
		priv, pub, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			return nil, fmt.Errorf("generate VAPID keys: %w", err)
		}
		cfg.VAPID.PublicKey, cfg.VAPID.PrivateKey = pub, priv
		slog.WarnContext(ctx, "VAPID keys are not set: using throwaway keys, push subscriptions break on restart")
	}

	tokens := &auth.JWTIssuer{Secret: []byte(cfg.JWTSecret), AccessTTL: cfg.AccessTTL, RefreshTTL: cfg.RefreshTTL, Clock: c}
	var mailer auth.Mailer = auth.LogMailer{}
	if cfg.SMTP.Addr != "" {
		mailer = auth.SMTPMailer{Addr: cfg.SMTP.Addr, From: cfg.SMTP.From, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password}
	}
	authSvc := auth.NewService(auth.Deps{
		Users:         st.users,
		Verifications: st.verifications,
		Identities:    st.identities,
		Guard:         auth.CounterGuard{Counter: st.counter},
		Tx:            st.tx,
		Hasher:        auth.BcryptHasher{},
		Tokens:        tokens,
		Mailer:        mailer,
		Refresh:       st.refresh,
		Clock:         c,
		VerifyURL:     cfg.PublicURL + BasePath + "/auth/verify",
	})
	rules := pets.DefaultRules()
	petSvc := pets.NewService(pets.Deps{
		Pets:      st.pets,
		Inventory: st.inventory,
		Tx:        st.tx,
		Brain:     brain.NewUtilityBrain(cfg.BrainSeed),
		Clock:     c,
		Rules:     rules,
		LowStat:   notifier,
		Publisher: petPublisher{broker: st.broker, rules: rules},
	})
	notifySvc := newNotifyService(cfg, st, c)

	if mem, ok := q.(*queue.Memory); ok {
		engine.New(engine.Deps{
			Pets:   st.pets,
			Notify: notifySvc,
			Queue:  mem,
			Users:  engine.UserNicknames{Users: st.users},
			Clock:  c,
		}).Register(mem)
	}

	return Router(tokens, Server{
		Handler:       auth.NewHandler(authSvc, oauthOptions(cfg, c)),
		PetsHandler:   pets.NewHandler(petSvc, c),
		NotifyHandler: notify.NewHandler(notifySvc),
	}, RouterOptions{
		Counter: st.counter,
		Limit:   ratelimit.Limit{Max: cfg.RateLimitPerMinute, Window: time.Minute},
		Origins: []string{cfg.PublicURL},
	})
}

func (a *App) queue(st storage) queue.Enqueuer {
	if st.redis != nil {
		e := queue.NewAsynqEnqueuer(st.redis)
		a.addCloser(e.Close)
		return e
	}
	slog.Warn("REDIS_URL is not set: background tasks run inside the API process")
	m := queue.NewMemory(nil, nil)
	a.addCloser(m.Close)
	return m
}

func newNotifyService(cfg config.Config, st storage, c clock.Clock) *notify.Service {
	var sender notify.Sender = notify.LogSender{}
	if cfg.VAPID.PublicKey != "" {
		sender = notify.WebPushSender{PublicKey: cfg.VAPID.PublicKey, PrivateKey: cfg.VAPID.PrivateKey, Subject: cfg.VAPID.Subject}
	}
	return notify.NewService(notify.Deps{
		Subscriptions: st.subscriptions,
		Preferences:   st.preferences,
		Counter:       st.counter,
		Sender:        sender,
		Clock:         c,
		PublicKey:     cfg.VAPID.PublicKey,
	})
}

func oauthOptions(cfg config.Config, c clock.Clock) auth.HandlerOptions {
	callback := func(provider string) string {
		return cfg.PublicURL + BasePath + "/auth/oauth/" + provider + "/callback"
	}
	var providers []auth.OAuthProvider
	if cfg.Google.Enabled() {
		providers = append(providers, auth.NewGoogleProvider(auth.OAuthConfig{
			ClientID: cfg.Google.ID, ClientSecret: cfg.Google.Secret, RedirectURL: callback(auth.ProviderGoogle),
		}))
	}
	if cfg.GitHub.Enabled() {
		providers = append(providers, auth.NewGitHubProvider(auth.OAuthConfig{
			ClientID: cfg.GitHub.ID, ClientSecret: cfg.GitHub.Secret, RedirectURL: callback(auth.ProviderGitHub),
		}))
	}
	return auth.HandlerOptions{
		SecureCookie: cfg.CookieSecure,
		AppURL:       cfg.PublicURL,
		Providers:    providers,
		StateSecret:  []byte(cfg.JWTSecret),
		Clock:        c,
	}
}

type petPublisher struct {
	broker realtime.Broker
	rules  pets.Rules
}

func (p petPublisher) PetUpdated(ctx context.Context, pet *pets.Pet) {
	realtime.PublishJSON(ctx, p.broker, realtime.PetTopic(pet.OwnerID), realtime.Message{
		Type: realtime.TypePetUpdated,
		Data: pets.NewPetView(pet, p.rules, pet.UpdatedAt),
	})
}

type Server struct {
	*auth.Handler
	*PetsHandler
	*NotifyHandler
}

type (
	PetsHandler   = pets.Handler
	NotifyHandler = notify.Handler
)

var _ oapi.ServerInterface = Server{}

type RouterOptions struct {
	Counter ratelimit.Counter
	Limit   ratelimit.Limit
	Origins []string
}

func Router(tokens httpx.TokenVerifier, srv oapi.ServerInterface, o RouterOptions) (http.Handler, error) {
	spec, err := oapi.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI spec: %w", err)
	}
	public := httpx.PublicOperations(spec, BasePath)
	isPublic := func(r *http.Request) bool {
		return public[r.Method+" "+chi.RouteContext(r.Context()).RoutePattern()]
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, httpx.SecurityHeaders, httpx.CSRF(o.Origins...))
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
		Middlewares: []oapi.MiddlewareFunc{
			httpx.ValidateRequests(spec, BasePath),
			httpx.RateLimit(httpx.RateLimitOptions{Counter: o.Counter, Limit: o.Limit}),
			httpx.Authenticate(tokens, isPublic),
		},
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			httpx.JSON(w, http.StatusBadRequest, httpx.ErrorBody{Error: "invalid", Message: err.Error()})
		},
	}), nil
}
