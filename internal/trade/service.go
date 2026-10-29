package trade

import (
	"context"
	"time"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/platform/tx"
)

type OfferRepository interface {
	Create(ctx context.Context, o *Offer) error
	ByIDForUpdate(ctx context.Context, id string) (*Offer, error)
	Save(ctx context.Context, o *Offer) error
	ListByUser(ctx context.Context, user string) ([]*Offer, error)
	ListPendingExpired(ctx context.Context, now time.Time) ([]string, error)
}

type IdempotencyStore interface {
	Get(ctx context.Context, key string) (*Offer, bool, error)
	Put(ctx context.Context, key string, result *Offer) error
}

type InventoryPort interface {
	ReserveItems(ctx context.Context, owner string, items map[string]int) error
	ReleaseItems(ctx context.Context, owner string, items map[string]int) error
	ExchangeItems(ctx context.Context, from, to string, give, want map[string]int) error
}

type FriendChecker interface {
	AreFriends(ctx context.Context, a, b string) (bool, error)
}

type Notifier interface {
	TradeChanged(ctx context.Context, userID string, o *Offer)
}

type Service struct {
	offers   OfferRepository
	idem     IdempotencyStore
	inv      InventoryPort
	friends  FriendChecker
	notifier Notifier
	tx       tx.Manager
	clock    clock.Clock
}

func NewService(offers OfferRepository, idem IdempotencyStore, inv InventoryPort, friends FriendChecker,
	n Notifier, t tx.Manager, c clock.Clock) *Service {
	return &Service{offers: offers, idem: idem, inv: inv, friends: friends, notifier: n, tx: t, clock: c}
}

func (s *Service) Create(ctx context.Context, from, to string, give, want map[string]int) (*Offer, error) {
	return nil, apperr.ErrNotImplemented
}

func (s *Service) Accept(ctx context.Context, user, offerID, idemKey string) (*Offer, error) {
	return nil, apperr.ErrNotImplemented
}

func (s *Service) Decline(ctx context.Context, user, offerID string) (*Offer, error) {
	return nil, apperr.ErrNotImplemented
}

func (s *Service) Cancel(ctx context.Context, user, offerID string) (*Offer, error) {
	return nil, apperr.ErrNotImplemented
}

func (s *Service) ExpireDue(ctx context.Context) (int, error) {
	return 0, apperr.ErrNotImplemented
}

func (s *Service) History(ctx context.Context, user string) ([]*Offer, error) {
	return nil, apperr.ErrNotImplemented
}
