package postgres

import (
	"context"
	"fmt"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/repository/postgres/sqlc"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

func (s *Store) ListReviews(ctx context.Context, query usecase.ReviewListQuery) ([]domain.Review, error) {
	switch query.Order {
	case usecase.ReviewOrderCreatedAt:
		return s.listReviewsByCreatedAt(ctx, query)
	case usecase.ReviewOrderScore:
		return s.listReviewsByScore(ctx, query)
	default:
		return nil, fmt.Errorf("unsupported review order %q", query.Order)
	}
}

func (s *Store) listReviewsByCreatedAt(ctx context.Context, query usecase.ReviewListQuery) ([]domain.Review, error) {
	albumID, err := idToPG(query.AlbumID)
	if err != nil {
		return nil, err
	}
	params := sqlc.ListReviewsByCreatedAtParams{AlbumID: albumID, RowLimit: int32(query.Limit)}
	if query.Cursor != nil {
		params.CursorCreatedAt = timeToPG(query.Cursor.CreatedAt)
		params.CursorID, err = idToPG(query.Cursor.ID)
		if err != nil {
			return nil, err
		}
	}
	rows, err := s.queries.ListReviewsByCreatedAt(ctx, params)
	if err != nil {
		return nil, err
	}
	return reviewsFromQueryRows(rows, func(row sqlc.ListReviewsByCreatedAtRow) reviewQueryRow {
		return reviewQueryRow(row)
	})
}

func (s *Store) listReviewsByScore(ctx context.Context, query usecase.ReviewListQuery) ([]domain.Review, error) {
	albumID, err := idToPG(query.AlbumID)
	if err != nil {
		return nil, err
	}
	params := sqlc.ListReviewsByScoreParams{AlbumID: albumID, RowLimit: int32(query.Limit)}
	if query.Cursor != nil {
		value := int16(query.Cursor.Score)
		params.CursorScore = &value
		params.CursorID, err = idToPG(query.Cursor.ID)
		if err != nil {
			return nil, err
		}
	}
	rows, err := s.queries.ListReviewsByScore(ctx, params)
	if err != nil {
		return nil, err
	}
	return reviewsFromQueryRows(rows, func(row sqlc.ListReviewsByScoreRow) reviewQueryRow {
		return reviewQueryRow(row)
	})
}
