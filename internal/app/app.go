package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/voloshmiak/tamagochi/internal/auth"
	"github.com/voloshmiak/tamagochi/internal/brain"
	"github.com/voloshmiak/tamagochi/internal/config"
	"github.com/voloshmiak/tamagochi/internal/pets"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/httpx"
	"github.com/voloshmiak/tamagochi/internal/platform/tx"
)

func New(cfg config.Config, c clock.Clock) http.Handler {
	tokens := &auth.JWTIssuer{Secret: []byte(cfg.JWTSecret), AccessTTL: cfg.AccessTTL, RefreshTTL: cfg.RefreshTTL, Clock: c}
	authSvc := auth.NewService(auth.NewMemoryUserRepository(), auth.BcryptHasher{}, tokens, auth.LogMailer{}, c)

	petSvc := pets.NewService(pets.Deps{
		Pets:      pets.NewMemoryPetRepository(),
		Inventory: pets.NewMemoryInventoryRepository(),
		Tx:        &tx.Local{},
		Brain:     brain.NewUtilityBrain(cfg.BrainSeed),
		Clock:     c,
	})

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	authH := auth.NewHandler(authSvc, cfg.SecureCookies())
	r.Route("/api/v1", func(r chi.Router) {
		authH.PublicRoutes(r)
		r.Group(func(r chi.Router) {
			r.Use(httpx.Authenticate(tokens))
			authH.Routes(r)
			pets.NewHandler(petSvc, c).Routes(r)
		})
	})
	return r
}
