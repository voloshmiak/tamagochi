package config

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env        string
	Addr       string
	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	BrainSeed  uint64
}

func Load() (Config, error) {
	var errs []error
	c := Config{
		Env:        cmp.Or(os.Getenv("APP_ENV"), "dev"),
		Addr:       cmp.Or(os.Getenv("HTTP_ADDR"), ":8080"),
		JWTSecret:  os.Getenv("JWT_SECRET"),
		AccessTTL:  duration("ACCESS_TTL", 15*time.Minute, &errs),
		RefreshTTL: duration("REFRESH_TTL", 30*24*time.Hour, &errs),
		BrainSeed:  42,
	}
	if v := os.Getenv("BRAIN_SEED"); v != "" {
		seed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("BRAIN_SEED: %w", err))
		}
		c.BrainSeed = seed
	}
	if c.JWTSecret == "" {
		if c.Env != "dev" {
			errs = append(errs, errors.New("JWT_SECRET is required outside dev"))
		}
		c.JWTSecret = "dev-only-insecure-secret"
	}
	return c, errors.Join(errs...)
}

func (c Config) SecureCookies() bool { return c.Env != "dev" }

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
