package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	AlbumTextMaxLength = 200
	GenreMaxLength     = 50
	FirstRecordedYear  = 1877
)

func LatestReleaseYear(now time.Time) int {
	return now.UTC().Year() + 1
}

type Album struct {
	ID           ID
	Owner        User
	Artist       string
	Title        string
	ReleaseYear  int
	Genre        string
	AverageScore *float64
	ReviewCount  int64
	Version      int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AlbumDraft struct {
	Artist      string
	Title       string
	ReleaseYear int
	Genre       string
}

type AlbumPatch struct {
	Artist      *string
	Title       *string
	ReleaseYear *int
	Genre       *string
	Version     int64
}

func ValidateAlbumDraft(draft AlbumDraft, now time.Time) (AlbumDraft, error) {
	draft.Artist = strings.TrimSpace(draft.Artist)
	draft.Title = strings.TrimSpace(draft.Title)
	draft.Genre = strings.TrimSpace(draft.Genre)

	violations := make([]Violation, 0, 4)
	violations = append(violations, validateText("artist", draft.Artist, AlbumTextMaxLength)...)
	violations = append(violations, validateText("title", draft.Title, AlbumTextMaxLength)...)
	violations = append(violations, validateText("genre", draft.Genre, GenreMaxLength)...)
	latestReleaseYear := LatestReleaseYear(now)
	if draft.ReleaseYear < FirstRecordedYear || draft.ReleaseYear > latestReleaseYear {
		violations = append(violations, Violation{
			Field:       "releaseYear",
			Description: fmt.Sprintf("must be between %d and %d", FirstRecordedYear, latestReleaseYear),
		})
	}

	return draft, NewValidationError(violations...)
}

func ApplyAlbumPatch(album Album, patch AlbumPatch, now time.Time) (AlbumDraft, error) {
	violations := make([]Violation, 0, 2)
	if patch.Version < 1 {
		violations = append(violations, Violation{Field: "version", Description: "must be positive"})
	}
	if patch.Artist == nil && patch.Title == nil && patch.ReleaseYear == nil && patch.Genre == nil {
		violations = append(violations, Violation{Field: "body", Description: "must change at least one field"})
	}
	if err := NewValidationError(violations...); err != nil {
		return AlbumDraft{}, err
	}

	draft := AlbumDraft{
		Artist:      album.Artist,
		Title:       album.Title,
		ReleaseYear: album.ReleaseYear,
		Genre:       album.Genre,
	}
	if patch.Artist != nil {
		draft.Artist = *patch.Artist
	}
	if patch.Title != nil {
		draft.Title = *patch.Title
	}
	if patch.ReleaseYear != nil {
		draft.ReleaseYear = *patch.ReleaseYear
	}
	if patch.Genre != nil {
		draft.Genre = *patch.Genre
	}

	return ValidateAlbumDraft(draft, now)
}
