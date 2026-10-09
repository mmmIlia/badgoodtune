package usecase

import (
	"fmt"
	"strings"
	"time"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

type AlbumOrder string

const (
	AlbumOrderCreatedAt   AlbumOrder = "createdAt desc"
	AlbumOrderReleaseYear AlbumOrder = "releaseYear desc"
	AlbumOrderRating      AlbumOrder = "rating desc"
	albumPageResource                = "albums"
)

type AlbumListRequest struct {
	PageSize    int
	PageToken   string
	Search      string
	Genre       string
	ReleaseYear *int
	Order       AlbumOrder
}

type AlbumCursor struct {
	ID          domain.ID
	CreatedAt   time.Time
	ReleaseYear int
	Rating      float64
}

type AlbumListQuery struct {
	Limit       int
	Search      string
	Genre       string
	ReleaseYear *int
	Order       AlbumOrder
	Cursor      *AlbumCursor
}

type AlbumPage struct {
	Albums        []domain.Album
	NextPageToken string
}

func normalizeAlbumList(request AlbumListRequest, now time.Time) (AlbumListQuery, error) {
	pageSize, err := normalizePageSize(request.PageSize)
	if err != nil {
		return AlbumListQuery{}, err
	}
	request.Search = strings.TrimSpace(request.Search)
	request.Genre = strings.TrimSpace(request.Genre)
	if request.Order == "" {
		request.Order = AlbumOrderCreatedAt
	}
	if !request.Order.valid() {
		return AlbumListQuery{}, domain.NewValidationError(domain.Violation{Field: "orderBy", Description: "is not supported"})
	}
	latestReleaseYear := domain.LatestReleaseYear(now)
	if request.ReleaseYear != nil &&
		(*request.ReleaseYear < domain.FirstRecordedYear || *request.ReleaseYear > latestReleaseYear) {
		return AlbumListQuery{}, domain.NewValidationError(domain.Violation{Field: "releaseYear", Description: "is invalid"})
	}

	query := AlbumListQuery{
		Limit:       pageSize + 1,
		Search:      request.Search,
		Genre:       request.Genre,
		ReleaseYear: request.ReleaseYear,
		Order:       request.Order,
	}
	if request.PageToken == "" {
		return query, nil
	}

	token, err := decodePageToken(request.PageToken)
	if err != nil || token.Resource != albumPageResource || token.Fingerprint != albumFingerprint(query) {
		return AlbumListQuery{}, invalidPageToken()
	}
	cursor := AlbumCursor{ID: domain.ID(token.ID)}
	switch query.Order {
	case AlbumOrderCreatedAt:
		if token.CreatedAt == nil {
			return AlbumListQuery{}, invalidPageToken()
		}
		cursor.CreatedAt = token.CreatedAt.UTC()
	case AlbumOrderReleaseYear:
		if token.ReleaseYear == nil ||
			*token.ReleaseYear < domain.FirstRecordedYear ||
			*token.ReleaseYear > latestReleaseYear {
			return AlbumListQuery{}, invalidPageToken()
		}
		cursor.ReleaseYear = *token.ReleaseYear
	case AlbumOrderRating:
		if token.Rating == nil || *token.Rating < 0 || *token.Rating > domain.ReviewScoreMax {
			return AlbumListQuery{}, invalidPageToken()
		}
		cursor.Rating = *token.Rating
	}
	query.Cursor = &cursor
	return query, nil
}

func albumPage(query AlbumListQuery, rows []domain.Album) (AlbumPage, error) {
	pageSize := query.Limit - 1
	if len(rows) <= pageSize {
		return AlbumPage{Albums: rows}, nil
	}
	items := rows[:pageSize]
	last := items[len(items)-1]
	token := pageToken{
		Version:     pageTokenVersion,
		Resource:    albumPageResource,
		Fingerprint: albumFingerprint(query),
		ID:          last.ID.String(),
	}
	switch query.Order {
	case AlbumOrderCreatedAt:
		value := last.CreatedAt.UTC()
		token.CreatedAt = &value
	case AlbumOrderReleaseYear:
		value := last.ReleaseYear
		token.ReleaseYear = &value
	case AlbumOrderRating:
		value := float64(0)
		if last.AverageScore != nil {
			value = *last.AverageScore
		}
		token.Rating = &value
	}
	encoded, err := encodePageToken(token)
	if err != nil {
		return AlbumPage{}, fmt.Errorf("encode album page token: %w", err)
	}
	return AlbumPage{Albums: items, NextPageToken: encoded}, nil
}

func (order AlbumOrder) valid() bool {
	return order == AlbumOrderCreatedAt || order == AlbumOrderReleaseYear || order == AlbumOrderRating
}

func albumFingerprint(query AlbumListQuery) string {
	year := ""
	if query.ReleaseYear != nil {
		year = fmt.Sprint(*query.ReleaseYear)
	}
	return fingerprint(strings.ToLower(query.Search), strings.ToLower(query.Genre), year, string(query.Order))
}
