package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type albumRepositoryFake struct {
	album   domain.Album
	rows    []domain.Album
	query   usecase.AlbumListQuery
	updated domain.AlbumDraft
	deleted bool
}

type reviewRepositoryFake struct {
	review  domain.Review
	updated bool
	deleted bool
}

func (f *reviewRepositoryFake) CreateReview(_ context.Context, albumID, authorID domain.ID, draft domain.ReviewDraft, now time.Time) (domain.Review, error) {
	return domain.Review{ID: "created", AlbumID: albumID, Author: domain.User{ID: authorID}, Headline: draft.Headline, Body: draft.Body, Score: draft.Score, Version: 1, CreatedAt: now, UpdatedAt: now}, nil
}

func (f *reviewRepositoryFake) GetReview(_ context.Context, _, _ domain.ID) (domain.Review, error) {
	if f.review.ID.IsZero() {
		return domain.Review{}, domain.ErrNotFound
	}
	return f.review, nil
}

func (f *reviewRepositoryFake) ListReviews(_ context.Context, _ usecase.ReviewListQuery) ([]domain.Review, error) {
	return []domain.Review{f.review}, nil
}

func (f *reviewRepositoryFake) UpdateReview(_ context.Context, _, _ domain.ID, _ int64, draft domain.ReviewDraft, _ time.Time) (domain.Review, error) {
	f.updated = true
	f.review.Headline = draft.Headline
	f.review.Body = draft.Body
	f.review.Score = draft.Score
	f.review.Version++
	return f.review, nil
}

func (f *reviewRepositoryFake) DeleteReview(_ context.Context, _, _ domain.ID, _ int64) error {
	f.deleted = true
	return nil
}

func (f *albumRepositoryFake) CreateAlbum(_ context.Context, ownerID domain.ID, draft domain.AlbumDraft, now time.Time) (domain.Album, error) {
	return domain.Album{ID: "created", Owner: domain.User{ID: ownerID}, Artist: draft.Artist, Title: draft.Title, ReleaseYear: draft.ReleaseYear, Genre: draft.Genre, Version: 1, CreatedAt: now, UpdatedAt: now}, nil
}

func (f *albumRepositoryFake) GetAlbum(_ context.Context, _ domain.ID) (domain.Album, error) {
	if f.album.ID.IsZero() {
		return domain.Album{}, domain.ErrNotFound
	}
	return f.album, nil
}

func (f *albumRepositoryFake) ListAlbums(_ context.Context, query usecase.AlbumListQuery) ([]domain.Album, error) {
	f.query = query
	return f.rows, nil
}

func (f *albumRepositoryFake) UpdateAlbum(_ context.Context, _ domain.ID, _ int64, draft domain.AlbumDraft, _ time.Time) (domain.Album, error) {
	f.updated = draft
	f.album.Artist = draft.Artist
	f.album.Version++
	return f.album, nil
}

func (f *albumRepositoryFake) DeleteAlbum(_ context.Context, _ domain.ID, _ int64) error {
	f.deleted = true
	return nil
}

func TestAlbumServiceRejectsAnotherOwner(t *testing.T) {
	t.Parallel()

	repository := &albumRepositoryFake{album: domain.Album{
		ID: "album", Owner: domain.User{ID: "owner"}, Artist: "Massive Attack", Title: "Mezzanine", ReleaseYear: 1998, Genre: "Trip hop", Version: 2,
	}}
	service, err := usecase.NewAlbumService(repository, fixedClock{now: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	artist := "Changed"

	_, err = service.Update(context.Background(), "attacker", "album", domain.AlbumPatch{Artist: &artist, Version: 2})

	require.ErrorIs(t, err, domain.ErrPermissionDenied)
	assert.Empty(t, repository.updated.Artist)
}

func TestAlbumServiceRejectsStaleVersion(t *testing.T) {
	t.Parallel()

	repository := &albumRepositoryFake{album: domain.Album{
		ID: "album", Owner: domain.User{ID: "owner"}, Artist: "Burial", Title: "Untrue", ReleaseYear: 2007, Genre: "UK garage", Version: 4,
	}}
	service, err := usecase.NewAlbumService(repository, fixedClock{now: time.Now()})
	require.NoError(t, err)
	title := "Changed"

	_, err = service.Update(context.Background(), "owner", "album", domain.AlbumPatch{Title: &title, Version: 3})

	require.ErrorIs(t, err, domain.ErrPreconditionFailed)
}

func TestAlbumPaginationTokenIsBoundToFilters(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	repository := &albumRepositoryFake{rows: []domain.Album{
		{ID: "3", CreatedAt: now},
		{ID: "2", CreatedAt: now.Add(-time.Hour)},
		{ID: "1", CreatedAt: now.Add(-2 * time.Hour)},
	}}
	service, err := usecase.NewAlbumService(repository, fixedClock{now: now})
	require.NoError(t, err)

	page, err := service.List(context.Background(), usecase.AlbumListRequest{PageSize: 2, Search: "portishead"})
	require.NoError(t, err)
	require.Len(t, page.Albums, 2)
	require.NotEmpty(t, page.NextPageToken)

	_, err = service.List(context.Background(), usecase.AlbumListRequest{PageSize: 2, Search: "bjork", PageToken: page.NextPageToken})
	var validationErr *domain.ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "pageToken", validationErr.Violations[0].Field)
}

func TestPageSizeIsCappedBeforeRepository(t *testing.T) {
	t.Parallel()

	repository := &albumRepositoryFake{}
	service, err := usecase.NewAlbumService(repository, fixedClock{now: time.Now()})
	require.NoError(t, err)

	_, err = service.List(context.Background(), usecase.AlbumListRequest{PageSize: 10_000})
	require.NoError(t, err)
	assert.Equal(t, usecase.MaxPageSize+1, repository.query.Limit)
}

func TestAlbumListRejectsFutureReleaseYear(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.October, 2, 0, 0, 0, 0, time.UTC)
	repository := &albumRepositoryFake{}
	service, err := usecase.NewAlbumService(repository, fixedClock{now: now})
	require.NoError(t, err)
	releaseYear := domain.LatestReleaseYear(now) + 1

	_, err = service.List(context.Background(), usecase.AlbumListRequest{ReleaseYear: &releaseYear})

	var validationErr *domain.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "releaseYear", validationErr.Violations[0].Field)
}

func TestReviewServiceEnforcesOwnerAndVersion(t *testing.T) {
	t.Parallel()

	reviews := &reviewRepositoryFake{review: domain.Review{
		ID: "review", AlbumID: "album", Author: domain.User{ID: "author"}, Headline: "Original",
		Body: "Detailed review", Score: 7, Version: 3,
	}}
	albums := &albumRepositoryFake{album: domain.Album{ID: "album", Owner: domain.User{ID: "owner"}}}
	service, err := usecase.NewReviewService(reviews, albums, fixedClock{now: time.Now()})
	require.NoError(t, err)
	score := 8

	_, err = service.Update(context.Background(), "another-user", "album", "review", domain.ReviewPatch{Score: &score, Version: 3})
	require.ErrorIs(t, err, domain.ErrPermissionDenied)
	assert.False(t, reviews.updated)

	err = service.Delete(context.Background(), "author", "album", "review", 2)
	require.ErrorIs(t, err, domain.ErrPreconditionFailed)
	assert.False(t, reviews.deleted)
}
