package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
)

const maxBodyBytes = 1 << 20

type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func Decode(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: malformed JSON body: %v", apperr.ErrInvalid, err)
	}
	return nil
}

func Error(w http.ResponseWriter, r *http.Request, err error) {
	status, code := statusOf(err)
	var ra apperr.RetryAfter
	if errors.As(err, &ra) {
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(ra.RetryAfter().Seconds()))))
	}
	if status == http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "err", err)
		JSON(w, status, ErrorBody{Error: code, Message: "internal error"})
		return
	}
	JSON(w, status, ErrorBody{Error: code, Message: err.Error()})
}

func statusOf(err error) (int, string) {
	switch {
	case errors.Is(err, apperr.ErrInvalid):
		return http.StatusBadRequest, "invalid"
	case errors.Is(err, apperr.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized"
	case errors.Is(err, apperr.ErrForbidden):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, apperr.ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, apperr.ErrConflict):
		return http.StatusConflict, "conflict"
	case errors.Is(err, apperr.ErrTooEarly):
		return http.StatusTooManyRequests, "cooldown"
	case errors.Is(err, apperr.ErrNotImplemented):
		return http.StatusNotImplemented, "not_implemented"
	default:
		return http.StatusInternalServerError, "internal"
	}
}

type userKey struct{}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userKey{}, userID)
}

func UserID(ctx context.Context) string {
	id, _ := ctx.Value(userKey{}).(string)
	return id
}

type TokenVerifier interface {
	VerifyAccess(token string) (userID string, err error)
}

const AccessCookie = "access_token"

func Authenticate(v TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			if c, err := r.Cookie(AccessCookie); err == nil {
				token = c.Value
			} else if h, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
				token = h
			}
			userID, err := v.VerifyAccess(token)
			if token == "" || err != nil {
				Error(w, r, fmt.Errorf("%w: missing or invalid access token", apperr.ErrUnauthorized))
				return
			}
			next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), userID)))
		})
	}
}
