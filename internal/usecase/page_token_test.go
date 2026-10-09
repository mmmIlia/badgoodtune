package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

func TestDecodePageTokenRejectsInvalidID(t *testing.T) {
	t.Parallel()

	raw, err := encodePageToken(pageToken{Version: pageTokenVersion, Resource: albumPageResource, ID: "not-a-uuid"})
	require.NoError(t, err)

	_, err = decodePageToken(raw)
	assert.Error(t, err)
}

func TestPaginationRejectsOutOfRangeCursors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.October, 2, 0, 0, 0, 0, time.UTC)
	albumQuery := AlbumListQuery{Limit: DefaultPageSize + 1, Order: AlbumOrderReleaseYear}
	releaseYear := domain.LatestReleaseYear(now) + 1
	albumToken, err := encodePageToken(pageToken{
		Version:     pageTokenVersion,
		Resource:    albumPageResource,
		Fingerprint: albumFingerprint(albumQuery),
		ID:          "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		ReleaseYear: &releaseYear,
	})
	require.NoError(t, err)
	_, err = normalizeAlbumList(AlbumListRequest{Order: AlbumOrderReleaseYear, PageToken: albumToken}, now)
	assert.Error(t, err)

	reviewQuery := ReviewListQuery{
		AlbumID: "6ba7b811-9dad-11d1-80b4-00c04fd430c8",
		Limit:   DefaultPageSize + 1,
		Order:   ReviewOrderScore,
	}
	score := domain.ReviewScoreMax + 1
	reviewToken, err := encodePageToken(pageToken{
		Version:     pageTokenVersion,
		Resource:    reviewPageResource,
		Fingerprint: reviewFingerprint(reviewQuery),
		ID:          "6ba7b812-9dad-11d1-80b4-00c04fd430c8",
		Score:       &score,
	})
	require.NoError(t, err)
	_, err = normalizeReviewList(ReviewListRequest{
		AlbumID:   reviewQuery.AlbumID,
		Order:     reviewQuery.Order,
		PageToken: reviewToken,
	})
	assert.Error(t, err)
}
