-- +goose Up
CREATE TABLE push_subscriptions (
    endpoint   text        PRIMARY KEY,
    user_id    text        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    p256dh     text        NOT NULL,
    auth       text        NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX push_subscriptions_user_id_idx ON push_subscriptions (user_id);

CREATE TABLE notification_preferences (
    user_id        text    PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    disabled_kinds text[]  NOT NULL DEFAULT '{}',
    quiet_enabled  boolean NOT NULL DEFAULT false,
    quiet_from     integer NOT NULL CHECK (quiet_from BETWEEN 0 AND 1439),
    quiet_to       integer NOT NULL CHECK (quiet_to BETWEEN 0 AND 1439),
    timezone       text    NOT NULL,
    lang           text    NOT NULL
);

-- +goose Down
DROP TABLE notification_preferences;
DROP TABLE push_subscriptions;
