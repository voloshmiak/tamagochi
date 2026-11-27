-- name: UserIDByIdentity :one
SELECT user_id FROM user_identities WHERE provider = $1 AND subject = $2;

-- name: LinkIdentity :exec
INSERT INTO user_identities (provider, subject, user_id, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (provider, subject) DO NOTHING;
