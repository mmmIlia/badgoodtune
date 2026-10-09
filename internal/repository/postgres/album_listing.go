package postgres

import (
	"context"
	"fmt"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/repository/postgres/sqlc"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

func (s *Store) ListAlbums(ctx context.Context, query usecase.AlbumListQuery) ([]domain.Album, error) {
	switch query.Order {
	case usecase.AlbumOrderCreatedAt:
		return s.listAlbumsByCreatedAt(ctx, query)
	case usecase.AlbumOrderReleaseYear:
		return s.listAlbumsByReleaseYear(ctx, query)
	case usecase.AlbumOrderRating:
		return s.listAlbumsByRating(ctx, query)
	default:
		return nil, fmt.Errorf("unsupported album order %q", query.Order)
	}
}

func (s *Store) listAlbumsByCreatedAt(ctx context.Context, query usecase.AlbumListQuery) ([]domain.Album, error) {
	params := sqlc.ListAlbumsByCreatedAtParams{
		Search:      query.Search,
		Genre:       query.Genre,
		ReleaseYear: optionalYear(query.ReleaseYear),
		RowLimit:    int32(query.Limit),
	}
	if query.Cursor != nil {
		params.CursorCreatedAt = timeToPG(query.Cursor.CreatedAt)
		cursorID, err := idToPG(query.Cursor.ID)
		if err != nil {
			return nil, err
		}
		params.CursorID = cursorID
	}
	rows, err := s.queries.ListAlbumsByCreatedAt(ctx, params)
	if err != nil {
		return nil, err
	}
	return albumsFromQueryRows(rows, func(row sqlc.ListAlbumsByCreatedAtRow) albumQueryRow {
		return albumQueryRow(row)
	})
}

func (s *Store) listAlbumsByReleaseYear(ctx context.Context, query usecase.AlbumListQuery) ([]domain.Album, error) {
	params := sqlc.ListAlbumsByReleaseYearParams{
		Search:      query.Search,
		Genre:       query.Genre,
		ReleaseYear: optionalYear(query.ReleaseYear),
		RowLimit:    int32(query.Limit),
	}
	if query.Cursor != nil {
		value := int16(query.Cursor.ReleaseYear)
		params.CursorReleaseYear = &value
		cursorID, err := idToPG(query.Cursor.ID)
		if err != nil {
			return nil, err
		}
		params.CursorID = cursorID
	}
	rows, err := s.queries.ListAlbumsByReleaseYear(ctx, params)
	if err != nil {
		return nil, err
	}
	return albumsFromQueryRows(rows, func(row sqlc.ListAlbumsByReleaseYearRow) albumQueryRow {
		return albumQueryRow(row)
	})
}

func (s *Store) listAlbumsByRating(ctx context.Context, query usecase.AlbumListQuery) ([]domain.Album, error) {
	params := sqlc.ListAlbumsByRatingParams{
		Search:      query.Search,
		Genre:       query.Genre,
		ReleaseYear: optionalYear(query.ReleaseYear),
		RowLimit:    int32(query.Limit),
	}
	if query.Cursor != nil {
		value := query.Cursor.Rating
		params.CursorRating = &value
		cursorID, err := idToPG(query.Cursor.ID)
		if err != nil {
			return nil, err
		}
		params.CursorID = cursorID
	}
	rows, err := s.queries.ListAlbumsByRating(ctx, params)
	if err != nil {
		return nil, err
	}
	return albumsFromQueryRows(rows, func(row sqlc.ListAlbumsByRatingRow) albumQueryRow {
		return albumQueryRow(row)
	})
}
