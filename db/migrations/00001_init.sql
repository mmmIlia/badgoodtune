-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username text NOT NULL,
    password_hash text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT users_username_trimmed CHECK (username = btrim(username)),
    CONSTRAINT users_username_format CHECK (username ~ '^[A-Za-z0-9_]{3,32}$'),
    CONSTRAINT users_password_hash_not_empty CHECK (password_hash <> '')
);

CREATE UNIQUE INDEX users_username_ci_uq ON users (lower(username));

CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT sessions_token_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT sessions_expiry_after_creation CHECK (expires_at > created_at)
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE albums (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    artist text NOT NULL,
    title text NOT NULL,
    release_year smallint NOT NULL,
    genre text NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT albums_artist_valid CHECK (artist = btrim(artist) AND char_length(artist) BETWEEN 1 AND 200),
    CONSTRAINT albums_title_valid CHECK (title = btrim(title) AND char_length(title) BETWEEN 1 AND 200),
    CONSTRAINT albums_genre_valid CHECK (genre = btrim(genre) AND char_length(genre) BETWEEN 1 AND 50),
    CONSTRAINT albums_release_year_valid CHECK (release_year BETWEEN 1877 AND 9999),
    CONSTRAINT albums_version_positive CHECK (version > 0),
    CONSTRAINT albums_updated_after_created CHECK (updated_at >= created_at)
);

CREATE UNIQUE INDEX albums_identity_ci_uq ON albums (lower(artist), lower(title), release_year);
CREATE INDEX albums_created_page_idx ON albums (created_at DESC, id DESC);
CREATE INDEX albums_release_year_page_idx ON albums (release_year DESC, id DESC);
CREATE INDEX albums_owner_id_idx ON albums (owner_id);

CREATE TABLE reviews (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    album_id uuid NOT NULL REFERENCES albums (id) ON DELETE RESTRICT,
    author_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    headline text NOT NULL,
    body text NOT NULL,
    score smallint NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT reviews_headline_valid CHECK (headline = btrim(headline) AND char_length(headline) BETWEEN 1 AND 120),
    CONSTRAINT reviews_body_valid CHECK (body = btrim(body) AND char_length(body) BETWEEN 1 AND 10000),
    CONSTRAINT reviews_score_valid CHECK (score BETWEEN 1 AND 10),
    CONSTRAINT reviews_version_positive CHECK (version > 0),
    CONSTRAINT reviews_updated_after_created CHECK (updated_at >= created_at),
    CONSTRAINT reviews_one_per_author UNIQUE (album_id, author_id)
);

CREATE INDEX reviews_album_created_page_idx ON reviews (album_id, created_at DESC, id DESC);
CREATE INDEX reviews_album_score_page_idx ON reviews (album_id, score DESC, id DESC);
CREATE INDEX reviews_author_id_idx ON reviews (author_id);

-- +goose Down
DROP TABLE reviews;
DROP TABLE albums;
DROP TABLE sessions;
DROP TABLE users;
