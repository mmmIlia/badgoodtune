package usecase

import (
	"context"
	"time"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

type AlbumRepository interface {
	CreateAlbum(
		ctx context.Context,
		ownerID domain.ID,
		draft domain.AlbumDraft,
		now time.Time,
	) (domain.Album, error)
	GetAlbum(ctx context.Context, albumID domain.ID) (domain.Album, error)
	ListAlbums(ctx context.Context, query AlbumListQuery) ([]domain.Album, error)
	UpdateAlbum(
		ctx context.Context,
		albumID domain.ID,
		expectedVersion int64,
		draft domain.AlbumDraft,
		now time.Time,
	) (domain.Album, error)
	DeleteAlbum(ctx context.Context, albumID domain.ID, expectedVersion int64) error
}
