package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/repository/postgres/sqlc"
)

func (s *Store) CreateReview(
	ctx context.Context,
	albumID domain.ID,
	authorID domain.ID,
	draft domain.ReviewDraft,
	now time.Time,
) (domain.Review, error) {
	albumUUID, err := idToPG(albumID)
	if err != nil {
		return domain.Review{}, err
	}
	authorUUID, err := idToPG(authorID)
	if err != nil {
		return domain.Review{}, err
	}
	id, err := s.queries.CreateReview(ctx, sqlc.CreateReviewParams{
		AlbumID:   albumUUID,
		AuthorID:  authorUUID,
		Headline:  draft.Headline,
		Body:      draft.Body,
		Score:     int16(draft.Score),
		CreatedAt: timeToPG(now),
	})
	if err != nil {
		return domain.Review{}, mapWriteError(err)
	}
	return s.getReviewByUUID(ctx, albumUUID, id)
}

func (s *Store) GetReview(ctx context.Context, albumID, reviewID domain.ID) (domain.Review, error) {
	albumUUID, err := idToPG(albumID)
	if err != nil {
		return domain.Review{}, err
	}
	reviewUUID, err := idToPG(reviewID)
	if err != nil {
		return domain.Review{}, err
	}
	return s.getReviewByUUID(ctx, albumUUID, reviewUUID)
}

func (s *Store) getReviewByUUID(ctx context.Context, albumID, reviewID pgtype.UUID) (domain.Review, error) {
	row, err := s.queries.GetReview(ctx, sqlc.GetReviewParams{AlbumID: albumID, ID: reviewID})
	if err != nil {
		return domain.Review{}, mapNotFound(err)
	}
	return reviewFromQueryRow(reviewQueryRow(row))
}

func (s *Store) UpdateReview(
	ctx context.Context,
	albumID domain.ID,
	reviewID domain.ID,
	expectedVersion int64,
	draft domain.ReviewDraft,
	now time.Time,
) (domain.Review, error) {
	albumUUID, err := idToPG(albumID)
	if err != nil {
		return domain.Review{}, err
	}
	reviewUUID, err := idToPG(reviewID)
	if err != nil {
		return domain.Review{}, err
	}
	rows, err := s.queries.UpdateReview(ctx, sqlc.UpdateReviewParams{
		AlbumID:   albumUUID,
		ID:        reviewUUID,
		Headline:  draft.Headline,
		Body:      draft.Body,
		Score:     int16(draft.Score),
		UpdatedAt: timeToPG(now),
		Version:   expectedVersion,
	})
	if err != nil {
		return domain.Review{}, mapWriteError(err)
	}
	if rows == 0 {
		return domain.Review{}, domain.ErrPreconditionFailed
	}
	return s.getReviewByUUID(ctx, albumUUID, reviewUUID)
}

func (s *Store) DeleteReview(ctx context.Context, albumID, reviewID domain.ID, expectedVersion int64) error {
	albumUUID, err := idToPG(albumID)
	if err != nil {
		return err
	}
	reviewUUID, err := idToPG(reviewID)
	if err != nil {
		return err
	}
	rows, err := s.queries.DeleteReview(ctx, sqlc.DeleteReviewParams{
		AlbumID: albumUUID,
		ID:      reviewUUID,
		Version: expectedVersion,
	})
	if err != nil {
		return mapWriteError(err)
	}
	if rows == 0 {
		return domain.ErrPreconditionFailed
	}
	return nil
}
