package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"

	usersUsernameConstraint        = "users_username_ci_uq"
	albumsIdentityConstraint       = "albums_identity_ci_uq"
	reviewsAuthorConstraint        = "reviews_one_per_author"
	sessionsUserConstraint         = "sessions_user_id_fkey"
	albumsOwnerConstraint          = "albums_owner_id_fkey"
	reviewsAlbumConstraint         = "reviews_album_id_fkey"
	reviewsAuthorForeignConstraint = "reviews_author_id_fkey"
)

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func mapWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case uniqueViolation:
		switch pgErr.ConstraintName {
		case usersUsernameConstraint, albumsIdentityConstraint, reviewsAuthorConstraint:
			return domain.ErrConflict
		}
	case foreignKeyViolation:
		switch pgErr.ConstraintName {
		case sessionsUserConstraint, albumsOwnerConstraint, reviewsAlbumConstraint, reviewsAuthorForeignConstraint:
			return domain.ErrNotFound
		}
	}
	return err
}

func mapAlbumDeleteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation && pgErr.ConstraintName == reviewsAlbumConstraint {
		return domain.ErrAlbumHasReviews
	}
	return mapWriteError(err)
}
