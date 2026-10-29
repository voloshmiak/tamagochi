package social

import (
	"context"
	"time"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusAccepted Status = "accepted"
	StatusBlocked  Status = "blocked"
)

type Relation struct {
	From      string
	To        string
	Status    Status
	CreatedAt time.Time
}

type Repository interface {
	Get(ctx context.Context, a, b string) (Relation, bool, error)
	Put(ctx context.Context, r Relation) error
	Delete(ctx context.Context, a, b string) error
	ListFor(ctx context.Context, user string) ([]Relation, error)
}

type UserDirectory interface {
	IDByNickname(ctx context.Context, nickname string) (string, error)
}

type Notifier interface {
	FriendRequest(ctx context.Context, from, to string)
}

type Service struct {
	repo     Repository
	users    UserDirectory
	notifier Notifier
	clock    clock.Clock
}

func NewService(repo Repository, users UserDirectory, n Notifier, c clock.Clock) *Service {
	return &Service{repo: repo, users: users, notifier: n, clock: c}
}

func (s *Service) SendRequest(ctx context.Context, from, nickname string) (Relation, error) {
	return Relation{}, apperr.ErrNotImplemented
}

func (s *Service) Accept(ctx context.Context, user, from string) error {
	return apperr.ErrNotImplemented
}

func (s *Service) Decline(ctx context.Context, user, other string) error {
	return apperr.ErrNotImplemented
}

func (s *Service) Remove(ctx context.Context, user, friend string) error {
	return apperr.ErrNotImplemented
}

func (s *Service) Block(ctx context.Context, user, other string) error {
	return apperr.ErrNotImplemented
}

func (s *Service) AreFriends(ctx context.Context, a, b string) (bool, error) {
	return false, apperr.ErrNotImplemented
}

func (s *Service) Friends(ctx context.Context, user string) ([]string, error) {
	return nil, apperr.ErrNotImplemented
}
