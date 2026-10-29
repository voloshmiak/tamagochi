package replica

import (
	"context"

	"github.com/voloshmiak/tamagochi/internal/brain"
	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
)

const MaxRunes = 140

type Request struct {
	UserID   string
	PetName  string
	Reaction brain.Reaction
	Lang     string
}

type Prompt struct {
	System string
	User   string
}

type Provider interface {
	Generate(ctx context.Context, p Prompt) (string, error)
}

type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key, text string)
}

type Quota interface {
	Allow(ctx context.Context, userID string) bool
}

type Templates interface {
	Pick(req Request) string
}

type Source string

const (
	SourceCache    Source = "cache"
	SourceLLM      Source = "llm"
	SourceTemplate Source = "template"
)

type Result struct {
	Text   string `json:"text"`
	Source Source `json:"source"`
}

type Service struct {
	provider  Provider
	cache     Cache
	quota     Quota
	templates Templates
}

func NewService(p Provider, c Cache, q Quota, t Templates) *Service {
	return &Service{provider: p, cache: c, quota: q, templates: t}
}

func (s *Service) Reply(ctx context.Context, req Request) (Result, error) {
	return Result{}, apperr.ErrNotImplemented
}
