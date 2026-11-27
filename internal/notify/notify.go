package notify

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"time"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/ratelimit"
)

type Kind string

const (
	KindLowStat       Kind = "low_stat"
	KindFriendRequest Kind = "friend_request"
	KindTrade         Kind = "trade"
)

var Kinds = []Kind{KindLowStat, KindFriendRequest, KindTrade}

const (
	DailyLimit     = 5
	dailyKeyTTL    = 48 * time.Hour
	dailyKeyPrefix = "notify:daily:"
)

type Notification struct {
	UserID string `json:"userId"`
	Kind   Kind   `json:"kind"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	URL    string `json:"url"`
	Tag    string `json:"tag,omitempty"`
}

type Subscription struct {
	UserID    string    `json:"-"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	CreatedAt time.Time `json:"-"`
}

var (
	ErrGone          = errors.New("push subscription gone")
	ErrBadSubscribe  = fmt.Errorf("%w: push subscription must have an https endpoint and both keys", apperr.ErrInvalid)
	ErrNotConfigured = fmt.Errorf("%w: web push is not configured on the server", apperr.ErrConflict)
)

type Sender interface {
	Send(ctx context.Context, s Subscription, n Notification) error
}

type SubscriptionStore interface {
	Add(ctx context.Context, s Subscription) error
	Remove(ctx context.Context, userID, endpoint string) error
	RemoveEndpoint(ctx context.Context, endpoint string) error
	ByUser(ctx context.Context, userID string) ([]Subscription, error)
}

type PreferenceStore interface {
	Get(ctx context.Context, userID string) (Preferences, error)
	Save(ctx context.Context, userID string, p Preferences) error
}

type Result string

const (
	ResultSent       Result = "sent"
	ResultDisabled   Result = "disabled"
	ResultQuietHours Result = "quiet_hours"
	ResultLimited    Result = "daily_limit"
	ResultNoDevices  Result = "no_devices"
)

type Deps struct {
	Subscriptions SubscriptionStore
	Preferences   PreferenceStore
	Counter       ratelimit.Counter
	Sender        Sender
	Clock         clock.Clock
	PublicKey     string
}

type Service struct {
	d Deps
}

func NewService(d Deps) *Service { return &Service{d: d} }

func (s *Service) PublicKey() string { return s.d.PublicKey }

func (s *Service) Subscribe(ctx context.Context, sub Subscription) error {
	if s.d.PublicKey == "" {
		return ErrNotConfigured
	}
	u, err := url.Parse(sub.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || sub.P256dh == "" || sub.Auth == "" {
		return ErrBadSubscribe
	}
	sub.CreatedAt = s.d.Clock.Now()
	return s.d.Subscriptions.Add(ctx, sub)
}

func (s *Service) Unsubscribe(ctx context.Context, userID, endpoint string) error {
	return s.d.Subscriptions.Remove(ctx, userID, endpoint)
}

func (s *Service) Preferences(ctx context.Context, userID string) (Preferences, error) {
	return s.d.Preferences.Get(ctx, userID)
}

func (s *Service) SetPreferences(ctx context.Context, userID string, p Preferences) (Preferences, error) {
	p, err := p.normalize()
	if err != nil {
		return Preferences{}, err
	}
	if err := s.d.Preferences.Save(ctx, userID, p); err != nil {
		return Preferences{}, err
	}
	return p, nil
}

func (s *Service) Deliver(ctx context.Context, n Notification) (Result, error) {
	prefs, err := s.d.Preferences.Get(ctx, n.UserID)
	if err != nil {
		return "", err
	}
	if !prefs.Allows(n.Kind) {
		return ResultDisabled, nil
	}
	now := s.d.Clock.Now()
	if prefs.InQuietHours(now) {
		return ResultQuietHours, nil
	}
	subs, err := s.d.Subscriptions.ByUser(ctx, n.UserID)
	if err != nil {
		return "", err
	}
	if len(subs) == 0 {
		return ResultNoDevices, nil
	}
	key := dailyKey(n.UserID, prefs.LocalDay(now))
	w, err := s.d.Counter.Incr(ctx, key, dailyKeyTTL)
	if err != nil {
		return "", err
	}
	if w.Count > DailyLimit {
		return ResultLimited, nil
	}
	if err := s.sendAll(ctx, subs, n); err != nil {
		if derr := s.d.Counter.Decr(ctx, key); derr != nil {
			slog.WarnContext(ctx, "release daily push slot", "user", n.UserID, "err", derr)
		}
		return "", err
	}
	return ResultSent, nil
}

func (s *Service) sendAll(ctx context.Context, subs []Subscription, n Notification) error {
	var errs []error
	delivered := 0
	for _, sub := range subs {
		err := s.d.Sender.Send(ctx, sub, n)
		switch {
		case err == nil:
			delivered++
		case errors.Is(err, ErrGone):
			if err := s.d.Subscriptions.RemoveEndpoint(ctx, sub.Endpoint); err != nil {
				errs = append(errs, err)
			}
		default:
			errs = append(errs, err)
		}
	}
	if delivered == 0 && len(errs) > 0 {
		return errors.Join(errs...)
	}
	for _, err := range errs {
		slog.WarnContext(ctx, "push to one of the devices failed", "user", n.UserID, "err", err)
	}
	return nil
}

func dailyKey(userID, day string) string { return dailyKeyPrefix + userID + ":" + day }

func validKind(k Kind) bool { return slices.Contains(Kinds, k) }
