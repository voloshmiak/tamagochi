package notify

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/voloshmiak/tamagochi/internal/platform/pg"
	"github.com/voloshmiak/tamagochi/internal/platform/pg/pgdb"
)

type PgSubscriptions struct {
	Pool *pgxpool.Pool
}

func (r PgSubscriptions) Add(ctx context.Context, s Subscription) error {
	err := pg.Queries(ctx, r.Pool).UpsertPushSubscription(ctx, pgdb.UpsertPushSubscriptionParams{
		Endpoint:  s.Endpoint,
		UserID:    s.UserID,
		P256dh:    s.P256dh,
		Auth:      s.Auth,
		CreatedAt: s.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("save push subscription: %w", err)
	}
	return nil
}

func (r PgSubscriptions) Remove(ctx context.Context, userID, endpoint string) error {
	err := pg.Queries(ctx, r.Pool).DeleteUserPushSubscription(ctx, pgdb.DeleteUserPushSubscriptionParams{UserID: userID, Endpoint: endpoint})
	if err != nil {
		return fmt.Errorf("delete push subscription: %w", err)
	}
	return nil
}

func (r PgSubscriptions) RemoveEndpoint(ctx context.Context, endpoint string) error {
	if err := pg.Queries(ctx, r.Pool).DeletePushSubscription(ctx, endpoint); err != nil {
		return fmt.Errorf("delete push subscription: %w", err)
	}
	return nil
}

func (r PgSubscriptions) ByUser(ctx context.Context, userID string) ([]Subscription, error) {
	rows, err := pg.Queries(ctx, r.Pool).PushSubscriptionsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load push subscriptions: %w", err)
	}
	out := make([]Subscription, len(rows))
	for i, row := range rows {
		out[i] = Subscription{UserID: row.UserID, Endpoint: row.Endpoint, P256dh: row.P256dh, Auth: row.Auth, CreatedAt: row.CreatedAt}
	}
	return out, nil
}

type PgPreferences struct {
	Pool *pgxpool.Pool
}

func (r PgPreferences) Get(ctx context.Context, userID string) (Preferences, error) {
	row, err := pg.Queries(ctx, r.Pool).NotificationPreferences(ctx, userID)
	if pg.IsNoRows(err) {
		return DefaultPreferences(), nil
	}
	if err != nil {
		return Preferences{}, fmt.Errorf("load notification preferences: %w", err)
	}
	p := Preferences{
		Disabled:     make(map[Kind]bool, len(row.DisabledKinds)),
		QuietEnabled: row.QuietEnabled,
		QuietFrom:    int(row.QuietFrom),
		QuietTo:      int(row.QuietTo),
		Timezone:     row.Timezone,
		Lang:         row.Lang,
	}
	for _, k := range row.DisabledKinds {
		p.Disabled[Kind(k)] = true
	}
	return p, nil
}

func (r PgPreferences) Save(ctx context.Context, userID string, p Preferences) error {
	disabled := make([]string, 0, len(p.Disabled))
	for k, off := range p.Disabled {
		if off {
			disabled = append(disabled, string(k))
		}
	}
	slices.Sort(disabled)
	err := pg.Queries(ctx, r.Pool).UpsertNotificationPreferences(ctx, pgdb.UpsertNotificationPreferencesParams{
		UserID:        userID,
		DisabledKinds: disabled,
		QuietEnabled:  p.QuietEnabled,
		QuietFrom:     int32(p.QuietFrom),
		QuietTo:       int32(p.QuietTo),
		Timezone:      p.Timezone,
		Lang:          p.Lang,
	})
	if err != nil {
		return fmt.Errorf("save notification preferences: %w", err)
	}
	return nil
}
