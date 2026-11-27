-- +goose Up
CREATE TABLE users (
    id             text        PRIMARY KEY,
    email          text        NOT NULL,
    nickname       text        NOT NULL,
    password_hash  bytea       NOT NULL,
    email_verified boolean     NOT NULL DEFAULT false,
    created_at     timestamptz NOT NULL
);

CREATE UNIQUE INDEX users_email_key ON users (email);
CREATE UNIQUE INDEX users_nickname_key ON users (lower(nickname));

CREATE TABLE email_verifications (
    token_hash text        PRIMARY KEY,
    user_id    text        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);

CREATE INDEX email_verifications_user_id_idx ON email_verifications (user_id);

-- +goose Down
DROP TABLE email_verifications;
DROP TABLE users;
