package engine

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/voloshmiak/tamagochi/internal/auth"
	"github.com/voloshmiak/tamagochi/internal/notify"
	"github.com/voloshmiak/tamagochi/internal/pets"
	"github.com/voloshmiak/tamagochi/internal/platform/clock"
	"github.com/voloshmiak/tamagochi/internal/queue"
)

type Nicknames interface {
	Nickname(ctx context.Context, userID string) (string, error)
}

type UserNicknames struct {
	Users auth.UserRepository
}

func (n UserNicknames) Nickname(ctx context.Context, userID string) (string, error) {
	u, err := n.Users.ByID(ctx, userID)
	return u.Nickname, err
}

type Deps struct {
	Pets   pets.PetRepository
	Notify *notify.Service
	Queue  queue.Enqueuer
	Users  Nicknames
	Clock  clock.Clock
}

type Engine struct {
	d Deps
}

func New(d Deps) *Engine { return &Engine{d: d} }

func (e *Engine) Register(r queue.Registrar) {
	r.Handle(queue.TaskPing, e.ping)
	r.Handle(queue.TaskLowStat, e.lowStat)
	r.Handle(queue.TaskNotify, e.notify)
}

func (e *Engine) ping(ctx context.Context, t queue.Task) error {
	var p PingPayload
	if err := decode(t, &p); err != nil {
		return err
	}
	slog.InfoContext(ctx, "ping task processed", "id", t.ID, "attempt", t.Attempt, "latency", e.d.Clock.Now().Sub(p.SentAt))
	return nil
}

func (e *Engine) lowStat(ctx context.Context, t queue.Task) error {
	var p LowStatPayload
	if err := decode(t, &p); err != nil {
		return err
	}
	pet, err := e.d.Pets.ByOwner(ctx, p.OwnerID)
	if errors.Is(err, pets.ErrPetNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !sameVersion(pet.UpdatedAt, p.Version) {
		return nil
	}
	now := e.d.Clock.Now()
	pet.Recompute(now)
	low := pet.LowStats(pets.LowStatThreshold)
	if len(low) == 0 {
		wait, _, ok := pet.TimeToThreshold(pets.LowStatThreshold)
		if !ok {
			return nil
		}
		return e.d.Queue.Enqueue(ctx, LowStatTask(p.OwnerID, p.Version, now.Add(wait)))
	}
	prefs, err := e.d.Notify.Preferences(ctx, p.OwnerID)
	if err != nil {
		return err
	}
	return e.deliver(ctx, notify.LowStat(p.OwnerID, prefs.Lang, pet.Name, string(low[0])))
}

func sameVersion(a, b time.Time) bool {
	return a.Truncate(time.Microsecond).Equal(b.Truncate(time.Microsecond))
}

func (e *Engine) notify(ctx context.Context, t queue.Task) error {
	var p NotifyPayload
	if err := decode(t, &p); err != nil {
		return err
	}
	prefs, err := e.d.Notify.Preferences(ctx, p.UserID)
	if err != nil {
		return err
	}
	from, err := e.d.Users.Nickname(ctx, p.FromUserID)
	if errors.Is(err, auth.ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var n notify.Notification
	switch p.Kind {
	case notify.KindFriendRequest:
		n = notify.FriendRequest(p.UserID, prefs.Lang, from)
	case notify.KindTrade:
		n = notify.TradeChanged(p.UserID, prefs.Lang, from, p.Status, p.OfferID)
	default:
		return errors.Join(queue.ErrSkipRetry, errors.New("unknown notification kind "+string(p.Kind)))
	}
	return e.deliver(ctx, n)
}

func (e *Engine) deliver(ctx context.Context, n notify.Notification) error {
	res, err := e.d.Notify.Deliver(ctx, n)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "notification processed", "user", n.UserID, "kind", n.Kind, "result", res)
	return nil
}
