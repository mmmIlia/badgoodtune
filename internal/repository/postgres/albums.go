package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/repository/postgres/sqlc"
)

func (s *Store) CreateAlbum(
	ctx context.Context,
	ownerID domain.ID,
	draft domain.AlbumDraft,
	now time.Time,
) (domain.Album, error) {
	ownerUUID, err := idToPG(ownerID)
	if err != nil {
		return domain.Album{}, err
	}
	id, err := s.queries.CreateAlbum(ctx, sqlc.CreateAlbumParams{
		OwnerID:     ownerUUID,
		Artist:      draft.Artist,
		Title:       draft.Title,
		ReleaseYear: int16(draft.ReleaseYear),
		Genre:       draft.Genre,
		CreatedAt:   timeToPG(now),
	})
	if err != nil {
		return domain.Album{}, mapWriteError(err)
	}
	return s.getAlbumByUUID(ctx, id)
}

func (s *Store) GetAlbum(ctx context.Context, albumID domain.ID) (domain.Album, error) {
	id, err := idToPG(albumID)
	if err != nil {
		return domain.Album{}, err
	}
	return s.getAlbumByUUID(ctx, id)
}

func (s *Store) getAlbumByUUID(ctx context.Context, id pgtype.UUID) (domain.Album, error) {
	row, err := s.queries.GetAlbum(ctx, id)
	if err != nil {
		return domain.Album{}, mapNotFound(err)
	}
	return albumFromQueryRow(albumQueryRow(row))
}

func (s *Store) UpdateAlbum(
	ctx context.Context,
	albumID domain.ID,
	expectedVersion int64,
	draft domain.AlbumDraft,
	now time.Time,
) (domain.Album, error) {
	id, err := idToPG(albumID)
	if err != nil {
		return domain.Album{}, err
	}
	rows, err := s.queries.UpdateAlbum(ctx, sqlc.UpdateAlbumParams{
		ID:          id,
		Version:     expectedVersion,
		Artist:      draft.Artist,
		Title:       draft.Title,
		ReleaseYear: int16(draft.ReleaseYear),
		Genre:       draft.Genre,
		UpdatedAt:   timeToPG(now),
	})
	if err != nil {
		return domain.Album{}, mapWriteError(err)
	}
	if rows == 0 {
		return domain.Album{}, domain.ErrPreconditionFailed
	}
	return s.getAlbumByUUID(ctx, id)
}

func (s *Store) DeleteAlbum(ctx context.Context, albumID domain.ID, expectedVersion int64) error {
	id, err := idToPG(albumID)
	if err != nil {
		return err
	}
	rows, err := s.queries.DeleteAlbum(ctx, sqlc.DeleteAlbumParams{ID: id, Version: expectedVersion})
	if err != nil {
		return mapAlbumDeleteError(err)
	}
	if rows == 0 {
		return domain.ErrPreconditionFailed
	}
	return nil
}
