package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

func TestValidateRegistration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		username   string
		password   string
		wantName   string
		wantFields []string
	}{
		{name: "valid and trimmed", username: "  Miles_42 ", password: "correct horse battery", wantName: "Miles_42"},
		{name: "bad username and short password", username: "a!", password: "short", wantName: "a!", wantFields: []string{"username", "password"}},
		{name: "unicode password counts runes", username: "listener", password: strings.Repeat("я", 12), wantName: "listener"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotName, err := domain.ValidateRegistration(tt.username, tt.password)
			assert.Equal(t, tt.wantName, gotName)
			if len(tt.wantFields) == 0 {
				require.NoError(t, err)
				return
			}

			var validationErr *domain.ValidationError
			require.ErrorAs(t, err, &validationErr)
			fields := make([]string, 0, len(validationErr.Violations))
			for _, violation := range validationErr.Violations {
				fields = append(fields, violation.Field)
			}
			assert.ElementsMatch(t, tt.wantFields, fields)
		})
	}
}

func TestValidateAlbumDraft(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.October, 2, 0, 0, 0, 0, time.UTC)
	draft, err := domain.ValidateAlbumDraft(domain.AlbumDraft{
		Artist:      "  Portishead ",
		Title:       " Dummy  ",
		ReleaseYear: 1994,
		Genre:       " Trip hop ",
	}, now)
	require.NoError(t, err)
	assert.Equal(t, "Portishead", draft.Artist)
	assert.Equal(t, "Dummy", draft.Title)
	assert.Equal(t, "Trip hop", draft.Genre)

	_, err = domain.ValidateAlbumDraft(domain.AlbumDraft{ReleaseYear: 2028}, now)
	var validationErr *domain.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Len(t, validationErr.Violations, 4)
}

func TestApplyAlbumPatchRequiresAChangeAndVersion(t *testing.T) {
	t.Parallel()

	album := domain.Album{Artist: "Björk", Title: "Homogenic", ReleaseYear: 1997, Genre: "Electronic"}
	_, err := domain.ApplyAlbumPatch(album, domain.AlbumPatch{}, time.Now())

	var validationErr *domain.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Len(t, validationErr.Violations, 2)
}

func TestReviewValidationAndPatch(t *testing.T) {
	t.Parallel()

	review := domain.Review{Headline: "Original", Body: "Detailed text", Score: 7}
	score := 10
	draft, err := domain.ApplyReviewPatch(review, domain.ReviewPatch{Score: &score, Version: 3})
	require.NoError(t, err)
	assert.Equal(t, 10, draft.Score)
	assert.Equal(t, review.Headline, draft.Headline)

	invalidScore := 11
	_, err = domain.ApplyReviewPatch(review, domain.ReviewPatch{Score: &invalidScore, Version: 3})
	var validationErr *domain.ValidationError
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "score", validationErr.Violations[0].Field)
}
