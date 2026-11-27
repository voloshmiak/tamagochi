package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/voloshmiak/tamagochi/internal/api/oapi"
	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
	"github.com/voloshmiak/tamagochi/internal/platform/httpx"
)

const (
	refreshCookie = "refresh_token"
	refreshPath   = "/api/v1/auth"
)

type Handler struct {
	svc          *Service
	secureCookie bool
	appURL       string
}

func NewHandler(svc *Service, secureCookie bool, appURL string) *Handler {
	return &Handler{svc: svc, secureCookie: secureCookie, appURL: appURL}
}

func newUserView(u User) oapi.User {
	return oapi.User{
		Id:            u.ID,
		Email:         u.Email,
		Nickname:      u.Nickname,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req oapi.RegisterJSONRequestBody
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	u, err := h.svc.Register(r.Context(), RegisterInput{
		Email:    string(req.Email),
		Password: req.Password,
		Nickname: req.Nickname,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, newUserView(u))
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request, params oapi.VerifyEmailParams) {
	result := "1"
	if err := h.svc.VerifyEmail(r.Context(), params.Token); err != nil {
		if !errors.Is(err, apperr.ErrInvalid) {
			httpx.Error(w, r, err)
			return
		}
		result = "0"
	}
	http.Redirect(w, r, h.appURL+"/?verified="+result, http.StatusSeeOther)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req oapi.LoginJSONRequestBody
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

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookie)
	if err != nil {
		httpx.Error(w, r, ErrInvalidCredentials)
		return
	}
	t, err := h.svc.Refresh(r.Context(), c.Value)
	if err != nil {
		if errors.Is(err, apperr.ErrUnauthorized) {
			h.clearCookies(w)
		}
		httpx.Error(w, r, err)
		return
	}
	h.setCookies(w, t)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookie); err == nil {
		if err := h.svc.Logout(r.Context(), c.Value); err != nil {
			httpx.Error(w, r, err)
			return
		}
	}
	h.clearCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.User(r.Context(), httpx.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, newUserView(u))
}

func (h *Handler) clearCookies(w http.ResponseWriter) {
	for _, name := range []string{httpx.AccessCookie, refreshCookie} {
		http.SetCookie(w, h.cookie(name, "", time.Unix(0, 0)))
	}
}

func (h *Handler) setCookies(w http.ResponseWriter, t Tokens) {
	http.SetCookie(w, h.cookie(httpx.AccessCookie, t.Access, t.AccessExpires))
	http.SetCookie(w, h.cookie(refreshCookie, t.Refresh, t.RefreshExpires))
}

func (h *Handler) cookie(name, value string, expires time.Time) *http.Cookie {
	path := "/"
	if name == refreshCookie {
		path = refreshPath
	}
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Expires:  expires,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
	}
}
