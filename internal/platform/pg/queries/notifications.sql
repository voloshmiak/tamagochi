-- name: UpsertPushSubscription :exec
INSERT INTO push_subscriptions (endpoint, user_id, p256dh, auth, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (endpoint) DO UPDATE
SET user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth, created_at = EXCLUDED.created_at;

-- name: DeleteUserPushSubscription :exec
DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2;

-- name: DeletePushSubscription :exec
DELETE FROM push_subscriptions WHERE endpoint = $1;

-- name: PushSubscriptionsByUser :many
SELECT * FROM push_subscriptions WHERE user_id = $1 ORDER BY endpoint;

-- name: NotificationPreferences :one
SELECT * FROM notification_preferences WHERE user_id = $1;

-- name: UpsertNotificationPreferences :exec
INSERT INTO notification_preferences (user_id, disabled_kinds, quiet_enabled, quiet_from, quiet_to, timezone, lang)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id) DO UPDATE
SET disabled_kinds = EXCLUDED.disabled_kinds, quiet_enabled = EXCLUDED.quiet_enabled,
    quiet_from = EXCLUDED.quiet_from, quiet_to = EXCLUDED.quiet_to,
    timezone = EXCLUDED.timezone, lang = EXCLUDED.lang;
