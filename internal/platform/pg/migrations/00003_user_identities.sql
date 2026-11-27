-- +goose Up
CREATE TABLE user_identities (
    provider   text        NOT NULL,
    subject    text        NOT NULL,
    user_id    text        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (provider, subject)
);

CREATE INDEX user_identities_user_id_idx ON user_identities (user_id);

-- +goose Down
DROP TABLE user_identities;
