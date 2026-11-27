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

	PublicURL    string
	CookieSecure bool

	DatabaseURL    string
	MigrateOnStart bool
	RedisURL       string

	RateLimitPerMinute int

	SMTP   SMTP
	VAPID  VAPID
	Google OAuthClient
	GitHub OAuthClient

	WorkerConcurrency int
	MetricsAddr       string
}

type VAPID struct {
	PublicKey  string
	PrivateKey string
	Subject    string
}

type OAuthClient struct {
	ID     string
	Secret string
}

func (c OAuthClient) Enabled() bool { return c.ID != "" && c.Secret != "" }

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

		RateLimitPerMinute: integer("RATE_LIMIT_PER_MINUTE", 60, &errs),
		WorkerConcurrency:  integer("WORKER_CONCURRENCY", 10, &errs),
		MetricsAddr:        cmp.Or(os.Getenv("METRICS_ADDR"), ":9090"),
		VAPID: VAPID{
			PublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
			PrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
			Subject:    cmp.Or(os.Getenv("VAPID_SUBJECT"), "mailto:admin@tamagochi.local"),
		},
		Google: OAuthClient{ID: os.Getenv("GOOGLE_CLIENT_ID"), Secret: os.Getenv("GOOGLE_CLIENT_SECRET")},
		GitHub: OAuthClient{ID: os.Getenv("GITHUB_CLIENT_ID"), Secret: os.Getenv("GITHUB_CLIENT_SECRET")},
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
	if c.RateLimitPerMinute <= 0 {
		errs = append(errs, fmt.Errorf("RATE_LIMIT_PER_MINUTE must be positive, got %d", c.RateLimitPerMinute))
	}
	if c.WorkerConcurrency <= 0 {
		errs = append(errs, fmt.Errorf("WORKER_CONCURRENCY must be positive, got %d", c.WorkerConcurrency))
	}
	if (c.VAPID.PublicKey == "") != (c.VAPID.PrivateKey == "") {
		errs = append(errs, errors.New("VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY must be set together"))
	}
	if c.Env != "dev" && (c.DatabaseURL == "" || c.RedisURL == "") {
		errs = append(errs, errors.New("DATABASE_URL and REDIS_URL are required outside dev: API instances keep no state of their own"))
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

func integer(key string, def int, errs *[]error) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: %w", key, err))
		return def
	}
	return n
}
