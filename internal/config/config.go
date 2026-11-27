package config

import (
	"cmp"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// NFR-SEC-03: access token lives at most 15 minutes, refresh at most 30 days.
const (
	MaxAccessTTL  = 15 * time.Minute
	MaxRefreshTTL = 30 * 24 * time.Hour
)

type Config struct {
	Env        string
	Addr       string
	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	BrainSeed  uint64

	// PublicURL is the address users open in the browser, without a trailing
	// slash. Links in emails point to it.
	PublicURL string
	// CookieSecure sets the Secure attribute on session cookies. It is on
	// everywhere except APP_ENV=dev unless COOKIE_SECURE says otherwise.
	CookieSecure bool

	// DatabaseURL selects PostgreSQL; when empty the API keeps data in memory.
	DatabaseURL string
	// MigrateOnStart applies pending migrations when the API starts.
	MigrateOnStart bool
	// RedisURL selects Redis for refresh sessions; when empty they are kept in memory.
	RedisURL string

	SMTP SMTP
}

// SMTP configures outgoing email. When Addr is empty emails are only logged.
type SMTP struct {
	Addr     string
	From     string
	Username string
	Password string
}

func Load() (Config, error) {
	var errs []error
	c := Config{
		Env:            cmp.Or(os.Getenv("APP_ENV"), "dev"),
		Addr:           cmp.Or(os.Getenv("HTTP_ADDR"), ":8080"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		AccessTTL:      duration("ACCESS_TTL", 15*time.Minute, &errs),
		RefreshTTL:     duration("REFRESH_TTL", 30*24*time.Hour, &errs),
		BrainSeed:      42,
		PublicURL:      strings.TrimRight(cmp.Or(os.Getenv("PUBLIC_URL"), "http://localhost:5173"), "/"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		MigrateOnStart: boolean("MIGRATE_ON_START", true, &errs),
		RedisURL:       os.Getenv("REDIS_URL"),
		SMTP: SMTP{
			Addr:     os.Getenv("SMTP_ADDR"),
			From:     cmp.Or(os.Getenv("SMTP_FROM"), "Tamagochi <no-reply@tamagochi.local>"),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
		},
	}
	c.CookieSecure = boolean("COOKIE_SECURE", c.Env != "dev", &errs)
	if v := os.Getenv("BRAIN_SEED"); v != "" {
		seed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("BRAIN_SEED: %w", err))
		}
		c.BrainSeed = seed
	}
	checkTTL("ACCESS_TTL", c.AccessTTL, MaxAccessTTL, &errs)
	checkTTL("REFRESH_TTL", c.RefreshTTL, MaxRefreshTTL, &errs)
	if u, err := url.Parse(c.PublicURL); err != nil || u.Scheme == "" || u.Host == "" {
		errs = append(errs, fmt.Errorf("PUBLIC_URL must be an absolute URL, got %q", c.PublicURL))
	}
	if c.JWTSecret == "" {
		if c.Env != "dev" {
			errs = append(errs, errors.New("JWT_SECRET is required outside dev"))
		}
		c.JWTSecret = "dev-only-insecure-secret"
	}
	return c, errors.Join(errs...)
}

func checkTTL(key string, d, limit time.Duration, errs *[]error) {
	if d <= 0 || d > limit {
		*errs = append(*errs, fmt.Errorf("%s must be greater than 0 and at most %s, got %s", key, limit, d))
	}
}

func duration(key string, def time.Duration, errs *[]error) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %w", key, err))
	}
	return d
}

func boolean(key string, def bool, errs *[]error) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %w", key, err))
		return def
	}
	return b
}
