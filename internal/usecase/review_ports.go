package usecase

import (
	"context"
	"time"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

type AlbumReader interface {
	GetAlbum(ctx context.Context, albumID domain.ID) (domain.Album, error)
}

type ReviewRepository interface {
	CreateReview(
		ctx context.Context,
		albumID domain.ID,
		authorID domain.ID,
		draft domain.ReviewDraft,
		now time.Time,
	) (domain.Review, error)
	GetReview(ctx context.Context, albumID, reviewID domain.ID) (domain.Review, error)
	ListReviews(ctx context.Context, query ReviewListQuery) ([]domain.Review, error)
	UpdateReview(
		ctx context.Context,
		albumID domain.ID,
		reviewID domain.ID,
		expectedVersion int64,
		draft domain.ReviewDraft,
		now time.Time,
	) (domain.Review, error)
	DeleteReview(ctx context.Context, albumID, reviewID domain.ID, expectedVersion int64) error
}
