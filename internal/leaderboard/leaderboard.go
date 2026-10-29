package leaderboard

import (
	"context"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
)

const TopN = 100

type Entry struct {
	UserID string `json:"userId"`
	Score  int    `json:"score"`
	Rank   int    `json:"rank"`
}

type Store interface {
	Add(ctx context.Context, userID string, points int) error
	Ranked(ctx context.Context) ([]Entry, error)
}

type FriendLister interface {
	Friends(ctx context.Context, user string) ([]string, error)
}

type Service struct {
	store   Store
	friends FriendLister
}

func NewService(store Store, friends FriendLister) *Service {
	return &Service{store: store, friends: friends}
}

func (s *Service) AddPoints(ctx context.Context, userID string, points int) error {
	return apperr.ErrNotImplemented
}

func (s *Service) Global(ctx context.Context, user string, n int) ([]Entry, Entry, error) {
	return nil, Entry{}, apperr.ErrNotImplemented
}

func (s *Service) Friends(ctx context.Context, user string) ([]Entry, error) {
	return nil, apperr.ErrNotImplemented
}
