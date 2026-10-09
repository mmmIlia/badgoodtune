package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

type ReviewService struct {
	repository ReviewRepository
	albums     AlbumReader
	clock      Clock
}

func NewReviewService(repository ReviewRepository, albums AlbumReader, clock Clock) (*ReviewService, error) {
	if repository == nil {
		return nil, errors.New("review repository is required")
	}
	if albums == nil {
		return nil, errors.New("album repository is required")
	}
	if clock == nil {
		return nil, errors.New("clock is required")
	}
	return &ReviewService{repository: repository, albums: albums, clock: clock}, nil
}

func (s *ReviewService) Create(
	ctx context.Context,
	actorID, albumID domain.ID,
	input domain.ReviewDraft,
) (domain.Review, error) {
	if actorID.IsZero() {
		return domain.Review{}, domain.ErrUnauthenticated
	}
	draft, err := domain.ValidateReviewDraft(input)
	if err != nil {
		return domain.Review{}, err
	}
	if albumID.IsZero() {
		return domain.Review{}, domain.NewValidationError(domain.Violation{Field: "albumId", Description: "is required"})
	}
	if _, err := s.albums.GetAlbum(ctx, albumID); err != nil {
		return domain.Review{}, fmt.Errorf("get album for review: %w", err)
	}
	review, err := s.repository.CreateReview(ctx, albumID, actorID, draft, s.clock.Now().UTC())
	if err != nil {
		return domain.Review{}, fmt.Errorf("create review: %w", err)
	}
	return review, nil
}

func (s *ReviewService) Get(ctx context.Context, albumID, reviewID domain.ID) (domain.Review, error) {
	if albumID.IsZero() || reviewID.IsZero() {
		return domain.Review{}, domain.NewValidationError(domain.Violation{Field: "id", Description: "albumId and reviewId are required"})
	}
	review, err := s.repository.GetReview(ctx, albumID, reviewID)
	if err != nil {
		return domain.Review{}, fmt.Errorf("get review: %w", err)
	}
	return review, nil
}

func (s *ReviewService) List(ctx context.Context, request ReviewListRequest) (ReviewPage, error) {
	query, err := normalizeReviewList(request)
	if err != nil {
		return ReviewPage{}, err
	}
	if _, err := s.albums.GetAlbum(ctx, query.AlbumID); err != nil {
		return ReviewPage{}, fmt.Errorf("get album for review list: %w", err)
	}
	rows, err := s.repository.ListReviews(ctx, query)
	if err != nil {
		return ReviewPage{}, fmt.Errorf("list reviews: %w", err)
	}
	return reviewPage(query, rows)
}

func (s *ReviewService) Update(
	ctx context.Context,
	actorID, albumID, reviewID domain.ID,
	patch domain.ReviewPatch,
) (domain.Review, error) {
	if actorID.IsZero() {
		return domain.Review{}, domain.ErrUnauthenticated
	}
	review, err := s.Get(ctx, albumID, reviewID)
	if err != nil {
		return domain.Review{}, err
	}
	if review.Author.ID != actorID {
		return domain.Review{}, domain.ErrPermissionDenied
	}
	draft, err := domain.ApplyReviewPatch(review, patch)
	if err != nil {
		return domain.Review{}, err
	}
	if review.Version != patch.Version {
		return domain.Review{}, domain.ErrPreconditionFailed
	}
	updated, err := s.repository.UpdateReview(ctx, albumID, reviewID, patch.Version, draft, s.clock.Now().UTC())
	if err != nil {
		return domain.Review{}, fmt.Errorf("update review: %w", err)
	}
	return updated, nil
}

func (s *ReviewService) Delete(
	ctx context.Context,
	actorID, albumID, reviewID domain.ID,
	expectedVersion int64,
) error {
	if actorID.IsZero() {
		return domain.ErrUnauthenticated
	}
	if expectedVersion < 1 {
		return domain.NewValidationError(domain.Violation{Field: "version", Description: "must be positive"})
	}
	review, err := s.Get(ctx, albumID, reviewID)
	if err != nil {
		return err
	}
	if review.Author.ID != actorID {
		return domain.ErrPermissionDenied
	}
	if review.Version != expectedVersion {
		return domain.ErrPreconditionFailed
	}
	if err := s.repository.DeleteReview(ctx, albumID, reviewID, expectedVersion); err != nil {
		return fmt.Errorf("delete review: %w", err)
	}
	return nil
}
