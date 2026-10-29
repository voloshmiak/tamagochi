package notify

import (
	"context"
	"errors"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
)

type Kind string

const (
	KindLowStat       Kind = "low_stat"
	KindFriendRequest Kind = "friend_request"
	KindTrade         Kind = "trade"
)

const DailyLimit = 5

type Notification struct {
	UserID string `json:"userId"`
	Kind   Kind   `json:"kind"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	URL    string `json:"url"`
}

type Subscription struct {
	UserID   string `json:"-"`
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

type Preferences struct {
	Disabled  map[Kind]bool `json:"disabled"`
	QuietFrom int           `json:"quietFrom"`
	QuietTo   int           `json:"quietTo"`
}

var ErrGone = errors.New("push subscription gone")

type Sender interface {
	Send(ctx context.Context, s Subscription, n Notification) error
}

type Store interface {
	AddSubscription(ctx context.Context, s Subscription) error
	RemoveSubscription(ctx context.Context, userID, endpoint string) error
	Subscriptions(ctx context.Context, userID string) ([]Subscription, error)
	Preferences(ctx context.Context, userID string) (Preferences, error)
	SavePreferences(ctx context.Context, userID string, p Preferences) error
	IncrDaily(ctx context.Context, userID, day string) (int, error)
}

type Service struct {
	store  Store
	sender Sender
	clock  clock.Clock
}

func NewService(store Store, sender Sender, c clock.Clock) *Service {
	return &Service{store: store, sender: sender, clock: c}
}

func (s *Service) Subscribe(ctx context.Context, sub Subscription) error {
	return apperr.ErrNotImplemented
}

func (s *Service) Unsubscribe(ctx context.Context, userID, endpoint string) error {
	return apperr.ErrNotImplemented
}

func (s *Service) SetPreferences(ctx context.Context, userID string, p Preferences) error {
	return apperr.ErrNotImplemented
}

func (s *Service) Deliver(ctx context.Context, n Notification) (bool, error) {
	return false, apperr.ErrNotImplemented
}
