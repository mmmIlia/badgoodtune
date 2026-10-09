-- name: CreateAlbum :one
INSERT INTO albums (owner_id, artist, title, release_year, genre, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $6)
RETURNING id;

-- name: GetAlbum :one
SELECT
    a.id,
    a.owner_id,
    u.username AS owner_username,
    u.created_at AS owner_created_at,
    a.artist,
    a.title,
    a.release_year,
    a.genre,
    COALESCE(AVG(r.score), 0)::double precision AS average_score,
    COUNT(r.id)::bigint AS review_count,
    a.version,
    a.created_at,
    a.updated_at
FROM albums a
JOIN users u ON u.id = a.owner_id
LEFT JOIN reviews r ON r.album_id = a.id
WHERE a.id = $1
GROUP BY a.id, u.id;

-- name: ListAlbumsByCreatedAt :many
SELECT
    a.id,
    a.owner_id,
    u.username AS owner_username,
    u.created_at AS owner_created_at,
    a.artist,
    a.title,
    a.release_year,
    a.genre,
    COALESCE(AVG(r.score), 0)::double precision AS average_score,
    COUNT(r.id)::bigint AS review_count,
    a.version,
    a.created_at,
    a.updated_at
FROM albums a
JOIN users u ON u.id = a.owner_id
LEFT JOIN reviews r ON r.album_id = a.id
WHERE (sqlc.arg(search)::text = ''
       OR position(lower(sqlc.arg(search)::text) IN lower(a.artist)) > 0
       OR position(lower(sqlc.arg(search)::text) IN lower(a.title)) > 0)
  AND (sqlc.arg(genre)::text = '' OR lower(a.genre) = lower(sqlc.arg(genre)::text))
  AND (sqlc.narg(release_year)::smallint IS NULL OR a.release_year = sqlc.narg(release_year)::smallint)
  AND (sqlc.narg(cursor_created_at)::timestamptz IS NULL OR (a.created_at, a.id) < (sqlc.narg(cursor_created_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
GROUP BY a.id, u.id
ORDER BY a.created_at DESC, a.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListAlbumsByReleaseYear :many
SELECT
    a.id,
    a.owner_id,
    u.username AS owner_username,
    u.created_at AS owner_created_at,
    a.artist,
    a.title,
    a.release_year,
    a.genre,
    COALESCE(AVG(r.score), 0)::double precision AS average_score,
    COUNT(r.id)::bigint AS review_count,
    a.version,
    a.created_at,
    a.updated_at
FROM albums a
JOIN users u ON u.id = a.owner_id
LEFT JOIN reviews r ON r.album_id = a.id
WHERE (sqlc.arg(search)::text = ''
       OR position(lower(sqlc.arg(search)::text) IN lower(a.artist)) > 0
       OR position(lower(sqlc.arg(search)::text) IN lower(a.title)) > 0)
  AND (sqlc.arg(genre)::text = '' OR lower(a.genre) = lower(sqlc.arg(genre)::text))
  AND (sqlc.narg(release_year)::smallint IS NULL OR a.release_year = sqlc.narg(release_year)::smallint)
  AND (sqlc.narg(cursor_release_year)::smallint IS NULL OR (a.release_year, a.id) < (sqlc.narg(cursor_release_year)::smallint, sqlc.narg(cursor_id)::uuid))
GROUP BY a.id, u.id
ORDER BY a.release_year DESC, a.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListAlbumsByRating :many
WITH album_rows AS (
    SELECT
        a.id,
        a.owner_id,
        u.username AS owner_username,
        u.created_at AS owner_created_at,
        a.artist,
        a.title,
        a.release_year,
        a.genre,
        COALESCE(AVG(r.score), 0)::double precision AS average_score,
        COUNT(r.id)::bigint AS review_count,
        a.version,
        a.created_at,
        a.updated_at
    FROM albums a
    JOIN users u ON u.id = a.owner_id
    LEFT JOIN reviews r ON r.album_id = a.id
    WHERE (sqlc.arg(search)::text = ''
           OR position(lower(sqlc.arg(search)::text) IN lower(a.artist)) > 0
           OR position(lower(sqlc.arg(search)::text) IN lower(a.title)) > 0)
      AND (sqlc.arg(genre)::text = '' OR lower(a.genre) = lower(sqlc.arg(genre)::text))
      AND (sqlc.narg(release_year)::smallint IS NULL OR a.release_year = sqlc.narg(release_year)::smallint)
    GROUP BY a.id, u.id
)
SELECT *
FROM album_rows
WHERE sqlc.narg(cursor_rating)::double precision IS NULL
   OR (COALESCE(average_score, 0::double precision), id) < (sqlc.narg(cursor_rating)::double precision, sqlc.narg(cursor_id)::uuid)
ORDER BY COALESCE(average_score, 0::double precision) DESC, id DESC
LIMIT sqlc.arg(row_limit);

-- name: UpdateAlbum :execrows
UPDATE albums
SET artist = $3,
    title = $4,
    release_year = $5,
    genre = $6,
    version = version + 1,
    updated_at = $7
WHERE id = $1
  AND version = $2;

-- name: DeleteAlbum :execrows
DELETE FROM albums
WHERE id = $1
  AND version = $2;
