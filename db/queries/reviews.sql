-- name: CreateReview :one
INSERT INTO reviews (album_id, author_id, headline, body, score, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $6)
RETURNING id;

-- name: GetReview :one
SELECT
    r.id,
    r.album_id,
    r.author_id,
    u.username AS author_username,
    u.created_at AS author_created_at,
    r.headline,
    r.body,
    r.score,
    r.version,
    r.created_at,
    r.updated_at
FROM reviews r
JOIN users u ON u.id = r.author_id
WHERE r.album_id = $1
  AND r.id = $2;

-- name: ListReviewsByCreatedAt :many
SELECT
    r.id,
    r.album_id,
    r.author_id,
    u.username AS author_username,
    u.created_at AS author_created_at,
    r.headline,
    r.body,
    r.score,
    r.version,
    r.created_at,
    r.updated_at
FROM reviews r
JOIN users u ON u.id = r.author_id
WHERE r.album_id = sqlc.arg(album_id)
  AND (sqlc.narg(cursor_created_at)::timestamptz IS NULL OR (r.created_at, r.id) < (sqlc.narg(cursor_created_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY r.created_at DESC, r.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListReviewsByScore :many
SELECT
    r.id,
    r.album_id,
    r.author_id,
    u.username AS author_username,
    u.created_at AS author_created_at,
    r.headline,
    r.body,
    r.score,
    r.version,
    r.created_at,
    r.updated_at
FROM reviews r
JOIN users u ON u.id = r.author_id
WHERE r.album_id = sqlc.arg(album_id)
  AND (sqlc.narg(cursor_score)::smallint IS NULL OR (r.score, r.id) < (sqlc.narg(cursor_score)::smallint, sqlc.narg(cursor_id)::uuid))
ORDER BY r.score DESC, r.id DESC
LIMIT sqlc.arg(row_limit);

-- name: UpdateReview :execrows
UPDATE reviews
SET headline = $3,
    body = $4,
    score = $5,
    version = version + 1,
    updated_at = $6
WHERE album_id = $1
  AND id = $2
  AND version = $7;

-- name: DeleteReview :execrows
DELETE FROM reviews
WHERE album_id = $1
  AND id = $2
  AND version = $3;
