package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/voloshmiak/tamagochi/internal/notify"
	"github.com/voloshmiak/tamagochi/internal/pets"
	"github.com/voloshmiak/tamagochi/internal/platform/tx"
	"github.com/voloshmiak/tamagochi/internal/queue"
	"github.com/voloshmiak/tamagochi/internal/social"
	"github.com/voloshmiak/tamagochi/internal/trade"
)

type Notifier struct {
	Queue queue.Enqueuer
}

var (
	_ social.Notifier       = Notifier{}
	_ trade.Notifier        = Notifier{}
	_ pets.LowStatScheduler = Notifier{}
)

func (n Notifier) FriendRequest(ctx context.Context, from, to string) {
	n.afterCommit(ctx, NotifyTask(NotifyPayload{Kind: notify.KindFriendRequest, UserID: to, FromUserID: from}))
}

func (n Notifier) TradeChanged(ctx context.Context, userID string, o *trade.Offer) {
	other := o.FromUser
	if userID == o.FromUser {
		other = o.ToUser
	}
	n.afterCommit(ctx, NotifyTask(NotifyPayload{
		Kind:       notify.KindTrade,
		UserID:     userID,
		FromUserID: other,
		OfferID:    o.ID,
		Status:     string(o.Status),
	}))
}

func (n Notifier) ScheduleLowStatCheck(ctx context.Context, ownerID string, version, at time.Time) error {
	return n.Queue.Enqueue(ctx, LowStatTask(ownerID, version, at))
}

func (n Notifier) afterCommit(ctx context.Context, t queue.Task) {
	tx.AfterCommit(ctx, func(ctx context.Context) {
		if err := n.Queue.Enqueue(ctx, t); err != nil {
			slog.ErrorContext(ctx, "enqueue notification", "task", t.Type, "err", err)
		}
	})
}
