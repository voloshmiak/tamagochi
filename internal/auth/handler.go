package auth

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/voloshmiak/tamagochi/internal/platform/httpx"
)

const refreshCookie = "refresh_token"

type Handler struct {
	svc          *Service
	secureCookie bool
}

func NewHandler(svc *Service, secureCookie bool) *Handler {
	return &Handler{svc: svc, secureCookie: secureCookie}
}

func (h *Handler) PublicRoutes(r chi.Router) {
	r.Post("/auth/register", h.register)
	r.Post("/auth/login", h.login)
	r.Post("/auth/refresh", h.refresh)
	r.Post("/auth/logout", h.logout)
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/me", h.me)
}

type UserView struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Nickname  string    `json:"nickname"`
	CreatedAt time.Time `json:"createdAt"`
}

func newUserView(u User) UserView {
	return UserView{ID: u.ID, Email: u.Email, Nickname: u.Nickname, CreatedAt: u.CreatedAt}
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	u, err := h.svc.Register(r.Context(), RegisterInput(req))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, newUserView(u))
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	u, t, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	h.setCookies(w, t)
	httpx.JSON(w, http.StatusOK, newUserView(u))
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookie)
	if err != nil {
		httpx.Error(w, r, ErrInvalidCredentials)
		return
	}
	t, err := h.svc.Refresh(r.Context(), c.Value)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	h.setCookies(w, t)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) logout(w http.ResponseWriter, _ *http.Request) {
	for _, name := range []string{httpx.AccessCookie, refreshCookie} {
		http.SetCookie(w, h.cookie(name, "", time.Unix(0, 0)))
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.User(r.Context(), httpx.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, newUserView(u))
}

func (h *Handler) setCookies(w http.ResponseWriter, t Tokens) {
	http.SetCookie(w, h.cookie(httpx.AccessCookie, t.Access, t.AccessExpires))
	http.SetCookie(w, h.cookie(refreshCookie, t.Refresh, t.RefreshExpires))
}

func (h *Handler) cookie(name, value string, expires time.Time) *http.Cookie {
	path := "/"
	if name == refreshCookie {
		path = "/api/v1/auth"
	}
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Expires:  expires,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	}
}
