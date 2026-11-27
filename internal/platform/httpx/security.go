package httpx

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
	"github.com/voloshmiak/tamagochi/internal/platform/ratelimit"
)

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func isSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

func CSRF(allowedOrigins ...string) func(http.Handler) http.Handler {
	allowed := make([]string, 0, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if origin := originOf(o); origin != "" {
			allowed = append(allowed, origin)
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			switch r.Header.Get("Sec-Fetch-Site") {
			case "", "same-origin", "none":
			default:
				Error(w, r, fmt.Errorf("%w: cross-site request rejected", apperr.ErrForbidden))
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" && !slices.Contains(allowed, originOf(origin)) {
				Error(w, r, fmt.Errorf("%w: origin %q is not allowed", apperr.ErrForbidden, origin))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func originOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Scheme + "://" + u.Host)
}

type RateLimitOptions struct {
	Counter ratelimit.Counter
	Limit   ratelimit.Limit
	Key     func(r *http.Request) string
}

func RateLimit(o RateLimitOptions) func(http.Handler) http.Handler {
	if o.Key == nil {
		o.Key = ClientKey
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			win, err := ratelimit.Take(r.Context(), o.Counter, "rl:req:"+o.Key(r), o.Limit)
			if err != nil && !isLimit(err) {
				slog.WarnContext(r.Context(), "rate limiter unavailable, request let through", "err", err)
				next.ServeHTTP(w, r)
				return
			}
			h := w.Header()
			h.Set("X-RateLimit-Limit", strconv.Itoa(o.Limit.Max))
			h.Set("X-RateLimit-Remaining", strconv.Itoa(max(o.Limit.Max-win.Count, 0)))
			h.Set("X-RateLimit-Reset", strconv.Itoa(int(math.Ceil(win.ResetIn.Seconds()))))
			if err != nil {
				Error(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isLimit(err error) bool {
	var le *ratelimit.LimitError
	return errors.As(err, &le)
}

func ClientKey(r *http.Request) string {
	if id := UserID(r.Context()); id != "" {
		return "user:" + id
	}
	return "ip:" + clientIP(r)
}

func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return strings.Trim(r.RemoteAddr, "[]")
}
