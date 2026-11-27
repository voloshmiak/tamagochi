-- name: CreateUser :exec
INSERT INTO users (id, email, nickname, password_hash, email_verified, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: UserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UserByNickname :one
SELECT * FROM users WHERE lower(nickname) = lower(sqlc.arg(nickname));

-- name: MarkEmailVerified :execrows
UPDATE users SET email_verified = true WHERE id = $1;

-- name: CreateEmailVerification :exec
INSERT INTO email_verifications (token_hash, user_id, expires_at) VALUES ($1, $2, $3);

-- name: ConsumeEmailVerification :one
DELETE FROM email_verifications WHERE token_hash = $1
RETURNING token_hash, user_id, expires_at;
