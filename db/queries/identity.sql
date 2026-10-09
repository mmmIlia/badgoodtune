-- name: CreateUser :one
INSERT INTO users (username, password_hash, created_at, updated_at)
VALUES ($1, $2, $3, $3)
RETURNING id, username, created_at;

-- name: GetCredentialByUsername :one
SELECT id, username, password_hash, created_at
FROM users
WHERE lower(username) = lower($1);

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at, created_at)
VALUES ($1, $2, $3, $4);

-- name: GetUserBySession :one
SELECT u.id, u.username, u.created_at
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1
  AND s.expires_at > $2;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions
WHERE expires_at <= $1;
