package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/voloshmiak/tamagochi/internal/api/oapi"
	"github.com/voloshmiak/tamagochi/internal/platform/httpx"
	"github.com/voloshmiak/tamagochi/internal/platform/id"
)

const (
	oauthCookie     = "oauth_state"
	oauthCookiePath = "/api/v1/auth/oauth"
	oauthStateTTL   = 10 * time.Minute
)

type oauthState struct {
	provider string
	state    string
	verifier string
	expires  time.Time
}

func (h *Handler) ListAuthProviders(w http.ResponseWriter, r *http.Request) {
	names := slices.Sorted(maps.Keys(h.providers))
	out := oapi.AuthProviders{Providers: make([]oapi.AuthProvider, len(names))}
	for i, n := range names {
		out.Providers[i] = oapi.AuthProvider(n)
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handler) StartOAuth(w http.ResponseWriter, r *http.Request, provider oapi.Provider) {
	p, ok := h.providers[string(provider)]
	if !ok {
		httpx.Error(w, r, ErrUnknownProvider)
		return
	}
	st := oauthState{
		provider: p.Name(),
		state:    id.New(),
		verifier: oauth2.GenerateVerifier(),
		expires:  h.clock.Now().Add(oauthStateTTL),
	}
	http.SetCookie(w, h.stateCookie(h.signState(st), st.expires))
	http.Redirect(w, r, p.AuthCodeURL(st.state, st.verifier), http.StatusFound)
}

func (h *Handler) OauthCallback(w http.ResponseWriter, r *http.Request, provider oapi.Provider, params oapi.OauthCallbackParams) {
	http.SetCookie(w, h.stateCookie("", time.Unix(0, 0)))
	fail := func(reason string, err error) {
		if err != nil {
			slog.WarnContext(r.Context(), "oauth login failed", "provider", provider, "reason", reason, "err", err)
		}
		http.Redirect(w, r, h.appURL+"/?oauth="+reason, http.StatusSeeOther)
	}
	p, ok := h.providers[string(provider)]
	if !ok {
		fail("error", ErrUnknownProvider)
		return
	}
	if params.Error != nil {
		fail("denied", nil)
		return
	}
	c, err := r.Cookie(oauthCookie)
	if err != nil {
		fail("error", errors.New("state cookie missing"))
		return
	}
	st, ok := h.verifyState(c.Value)
	if !ok || st.provider != p.Name() || params.State == nil || !hmac.Equal([]byte(*params.State), []byte(st.state)) {
		fail("error", errors.New("state mismatch"))
		return
	}
	if params.Code == nil || *params.Code == "" {
		fail("error", errors.New("no authorization code"))
		return
	}
	ident, err := p.Identify(r.Context(), *params.Code, st.verifier)
	if err != nil {
		fail("error", err)
		return
	}
	_, t, _, err := h.svc.LoginWithIdentity(r.Context(), ident)
	if errors.Is(err, ErrNoVerifiedEmail) {
		fail("no_email", err)
		return
	}
	if err != nil {
		fail("error", err)
		return
	}
	h.setCookies(w, t)
	http.Redirect(w, r, h.appURL+"/", http.StatusSeeOther)
}

func (h *Handler) stateCookie(value string, expires time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     oauthCookie,
		Value:    value,
		Path:     oauthCookiePath,
		Expires:  expires,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	}
}

func (h *Handler) signState(st oauthState) string {
	raw := strings.Join([]string{st.provider, st.state, st.verifier, strconv.FormatInt(st.expires.Unix(), 10)}, "|")
	payload := base64.RawURLEncoding.EncodeToString([]byte(raw))
	return payload + "." + h.mac(payload)
}

func (h *Handler) verifyState(v string) (oauthState, bool) {
	payload, sig, ok := strings.Cut(v, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(h.mac(payload))) {
		return oauthState{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return oauthState{}, false
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 4 {
		return oauthState{}, false
	}
	exp, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || !time.Unix(exp, 0).After(h.clock.Now()) {
		return oauthState{}, false
	}
	return oauthState{provider: parts[0], state: parts[1], verifier: parts[2], expires: time.Unix(exp, 0)}, true
}

func (h *Handler) mac(payload string) string {
	m := hmac.New(sha256.New, h.stateSecret)
	m.Write([]byte("oauth-state:" + payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
