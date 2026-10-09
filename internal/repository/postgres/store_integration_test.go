//go:build integration

package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	postgresadapter "github.com/mmmIlia/badgoodtune/internal/repository/postgres"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

func TestStoreCRUDAndConstraints(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()

	database, err := sql.Open("pgx", databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	require.NoError(t, postgresadapter.Migrate(ctx, database))
	_, err = database.ExecContext(ctx, "TRUNCATE reviews, albums, sessions, users CASCADE")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	store, err := postgresadapter.NewStore(pool)
	require.NoError(t, err)

	now := time.Date(2026, time.October, 2, 12, 0, 0, 0, time.UTC)
	user, err := store.CreateUserWithSession(ctx, "listener", "argon2-hash", usecase.NewSession{
		TokenHash: make([]byte, 32), CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	require.NoError(t, err)
	require.NotEmpty(t, user.ID)

	_, err = store.CreateUserWithSession(ctx, "LISTENER", "another-hash", usecase.NewSession{
		TokenHash: bytesOf(1, 32), CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	require.ErrorIs(t, err, domain.ErrConflict)

	authenticated, err := store.GetUserBySession(ctx, make([]byte, 32), now)
	require.NoError(t, err)
	assert.Equal(t, user.ID, authenticated.ID)
	_, err = store.GetUserBySession(ctx, make([]byte, 32), now.Add(2*time.Hour))
	require.ErrorIs(t, err, domain.ErrNotFound)
	err = store.CreateSession(ctx, user.ID, usecase.NewSession{
		TokenHash: bytesOf(2, 32), CreatedAt: now.Add(2 * time.Hour), ExpiresAt: now.Add(3 * time.Hour),
	})
	require.NoError(t, err)
	var sessionCount int
	require.NoError(t, database.QueryRowContext(ctx, "SELECT count(*) FROM sessions").Scan(&sessionCount))
	assert.Equal(t, 1, sessionCount)

	album, err := store.CreateAlbum(ctx, user.ID, domain.AlbumDraft{
		Artist: "Portishead", Title: "Dummy", ReleaseYear: 1994, Genre: "Trip hop",
	}, now)
	require.NoError(t, err)
	assert.Nil(t, album.AverageScore)
	assert.EqualValues(t, 0, album.ReviewCount)

	_, err = store.CreateAlbum(ctx, user.ID, domain.AlbumDraft{
		Artist: "PORTISHEAD", Title: "dummy", ReleaseYear: 1994, Genre: "Trip hop",
	}, now)
	require.ErrorIs(t, err, domain.ErrConflict)
	rows, err := store.ListAlbums(ctx, usecase.AlbumListQuery{
		Limit:  10,
		Search: "%",
		Order:  usecase.AlbumOrderCreatedAt,
	})
	require.NoError(t, err)
	assert.Empty(t, rows)

	review, err := store.CreateReview(ctx, album.ID, user.ID, domain.ReviewDraft{
		Headline: "A landmark", Body: "A complete and careful review.", Score: 9,
	}, now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 1, int(review.Version))

	_, err = store.CreateReview(ctx, album.ID, user.ID, domain.ReviewDraft{
		Headline: "Duplicate", Body: "Must not be stored.", Score: 8,
	}, now.Add(2*time.Minute))
	require.ErrorIs(t, err, domain.ErrConflict)

	album, err = store.GetAlbum(ctx, album.ID)
	require.NoError(t, err)
	require.NotNil(t, album.AverageScore)
	assert.Equal(t, 9.0, *album.AverageScore)
	assert.EqualValues(t, 1, album.ReviewCount)

	review, err = store.UpdateReview(ctx, album.ID, review.ID, review.Version, domain.ReviewDraft{
		Headline: "Still a landmark", Body: "Updated after another careful listen.", Score: 8,
	}, now.Add(2*time.Minute))
	require.NoError(t, err)
	assert.EqualValues(t, 2, review.Version)
	assert.Equal(t, 8, review.Score)
	_, err = store.UpdateReview(ctx, album.ID, review.ID, 1, domain.ReviewDraft{
		Headline: review.Headline, Body: review.Body, Score: 7,
	}, now.Add(3*time.Minute))
	require.ErrorIs(t, err, domain.ErrPreconditionFailed)
	album, err = store.GetAlbum(ctx, album.ID)
	require.NoError(t, err)
	require.NotNil(t, album.AverageScore)
	assert.Equal(t, 8.0, *album.AverageScore)

	_, err = store.UpdateAlbum(ctx, album.ID, album.Version+1, domain.AlbumDraft{
		Artist: album.Artist, Title: album.Title, ReleaseYear: album.ReleaseYear, Genre: album.Genre,
	}, now.Add(4*time.Minute))
	require.ErrorIs(t, err, domain.ErrPreconditionFailed)

	err = store.DeleteAlbum(ctx, album.ID, album.Version)
	require.ErrorIs(t, err, domain.ErrAlbumHasReviews)

	err = store.DeleteReview(ctx, album.ID, review.ID, review.Version)
	require.NoError(t, err)
	err = store.DeleteAlbum(ctx, album.ID, album.Version)
	require.NoError(t, err)

	_, err = store.GetAlbum(ctx, album.ID)
	require.True(t, errors.Is(err, domain.ErrNotFound))
}

func bytesOf(value byte, size int) []byte {
	result := make([]byte, size)
	for index := range result {
		result[index] = value
	}
	return result
}
