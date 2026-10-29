package trade

import (
	"fmt"
	"maps"
	"time"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusDeclined  Status = "declined"
	StatusCancelled Status = "cancelled"
	StatusExpired   Status = "expired"
)

const OfferTTL = 24 * time.Hour

var (
	ErrOfferNotFound     = fmt.Errorf("%w: trade offer", apperr.ErrNotFound)
	ErrInvalidTransition = fmt.Errorf("%w: offer is already closed", apperr.ErrConflict)
)

type Offer struct {
	ID         string         `json:"id"`
	FromUser   string         `json:"fromUser"`
	ToUser     string         `json:"toUser"`
	Give       map[string]int `json:"give"`
	Want       map[string]int `json:"want"`
	Status     Status         `json:"status"`
	FailReason string         `json:"failReason,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
	ExpiresAt  time.Time      `json:"expiresAt"`
	ClosedAt   time.Time      `json:"closedAt,omitzero"`
}

func NewOffer(id, from, to string, give, want map[string]int, now time.Time) (*Offer, error) {
	if from == to {
		return nil, fmt.Errorf("%w: cannot trade with yourself", apperr.ErrInvalid)
	}
	if len(give) == 0 && len(want) == 0 {
		return nil, fmt.Errorf("%w: offer is empty", apperr.ErrInvalid)
	}
	for _, items := range []map[string]int{give, want} {
		for itemID, n := range items {
			if n <= 0 {
				return nil, fmt.Errorf("%w: quantity of %q must be positive", apperr.ErrInvalid, itemID)
			}
		}
	}
	return &Offer{
		ID: id, FromUser: from, ToUser: to,
		Give: maps.Clone(give), Want: maps.Clone(want),
		Status: StatusPending, CreatedAt: now, ExpiresAt: now.Add(OfferTTL),
	}, nil
}

func (o *Offer) Clone() *Offer {
	c := *o
	c.Give, c.Want = maps.Clone(o.Give), maps.Clone(o.Want)
	return &c
}

func (o *Offer) IsExpired(now time.Time) bool { return !now.Before(o.ExpiresAt) }

func (o *Offer) close(to Status, now time.Time) error {
	if o.Status != StatusPending {
		return fmt.Errorf("%w (status %s)", ErrInvalidTransition, o.Status)
	}
	o.Status, o.ClosedAt = to, now
	return nil
}

func (o *Offer) Complete(now time.Time) error { return o.close(StatusCompleted, now) }
func (o *Offer) Decline(now time.Time) error  { return o.close(StatusDeclined, now) }
func (o *Offer) Cancel(now time.Time) error   { return o.close(StatusCancelled, now) }
func (o *Offer) Expire(now time.Time) error   { return o.close(StatusExpired, now) }

func (o *Offer) Fail(reason string, now time.Time) error {
	if err := o.close(StatusFailed, now); err != nil {
		return err
	}
	o.FailReason = reason
	return nil
}
