package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

type AlbumService struct {
	repository AlbumRepository
	clock      Clock
	observer   AlbumObserver
}

func NewAlbumService(repository AlbumRepository, clock Clock, observer AlbumObserver) (*AlbumService, error) {
	if observer == nil {
		return nil, errors.New("album observer is required")
	}
	if repository == nil {
		return nil, errors.New("album repository is required")
	}
	if clock == nil {
		return nil, errors.New("clock is required")
	}
	return &AlbumService{repository: repository, clock: clock, observer: observer}, nil
}

func (s *AlbumService) Create(ctx context.Context, actorID domain.ID, input domain.AlbumDraft) (domain.Album, error) {
	if actorID.IsZero() {
		return domain.Album{}, domain.ErrUnauthenticated
	}
	now := s.clock.Now().UTC()
	draft, err := domain.ValidateAlbumDraft(input, now)
	if err != nil {
		return domain.Album{}, err
	}
	album, err := s.repository.CreateAlbum(ctx, actorID, draft, now)
	if err != nil {
		return domain.Album{}, fmt.Errorf("create album: %w", err)
	}
	s.observer.AlbumCreated()
	return album, nil
}

func (s *AlbumService) Get(ctx context.Context, albumID domain.ID) (domain.Album, error) {
	if albumID.IsZero() {
		return domain.Album{}, domain.NewValidationError(domain.Violation{Field: "albumId", Description: "is required"})
	}
	album, err := s.repository.GetAlbum(ctx, albumID)
	if err != nil {
		return domain.Album{}, fmt.Errorf("get album: %w", err)
	}
	return album, nil
}

func (s *AlbumService) List(ctx context.Context, request AlbumListRequest) (AlbumPage, error) {
	query, err := normalizeAlbumList(request, s.clock.Now())
	if err != nil {
		return AlbumPage{}, err
	}
	rows, err := s.repository.ListAlbums(ctx, query)
	if err != nil {
		return AlbumPage{}, fmt.Errorf("list albums: %w", err)
	}
	return albumPage(query, rows)
}

func (s *AlbumService) Update(
	ctx context.Context,
	actorID, albumID domain.ID,
	patch domain.AlbumPatch,
) (domain.Album, error) {
	if actorID.IsZero() {
		return domain.Album{}, domain.ErrUnauthenticated
	}
	album, err := s.Get(ctx, albumID)
	if err != nil {
		return domain.Album{}, err
	}
	if album.Owner.ID != actorID {
		return domain.Album{}, domain.ErrPermissionDenied
	}
	now := s.clock.Now().UTC()
	draft, err := domain.ApplyAlbumPatch(album, patch, now)
	if err != nil {
		return domain.Album{}, err
	}
	if album.Version != patch.Version {
		return domain.Album{}, domain.ErrPreconditionFailed
	}
	updated, err := s.repository.UpdateAlbum(ctx, albumID, patch.Version, draft, now)
	if err != nil {
		return domain.Album{}, fmt.Errorf("update album: %w", err)
	}
	return updated, nil
}

func (s *AlbumService) Delete(ctx context.Context, actorID, albumID domain.ID, expectedVersion int64) error {
	if actorID.IsZero() {
		return domain.ErrUnauthenticated
	}
	if expectedVersion < 1 {
		return domain.NewValidationError(domain.Violation{Field: "version", Description: "must be positive"})
	}
	album, err := s.Get(ctx, albumID)
	if err != nil {
		return err
	}
	if album.Owner.ID != actorID {
		return domain.ErrPermissionDenied
	}
	if album.Version != expectedVersion {
		return domain.ErrPreconditionFailed
	}
	if err := s.repository.DeleteAlbum(ctx, albumID, expectedVersion); err != nil {
		return fmt.Errorf("delete album: %w", err)
	}
	return nil
}
