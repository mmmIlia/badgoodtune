package usecase

import (
	"fmt"
	"time"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

type ReviewOrder string

const (
	ReviewOrderCreatedAt ReviewOrder = "createdAt desc"
	ReviewOrderScore     ReviewOrder = "score desc"
	reviewPageResource               = "reviews"
)

type ReviewListRequest struct {
	AlbumID   domain.ID
	PageSize  int
	PageToken string
	Order     ReviewOrder
}

type ReviewCursor struct {
	ID        domain.ID
	CreatedAt time.Time
	Score     int
}

type ReviewListQuery struct {
	AlbumID domain.ID
	Limit   int
	Order   ReviewOrder
	Cursor  *ReviewCursor
}

type ReviewPage struct {
	Reviews       []domain.Review
	NextPageToken string
}

func normalizeReviewList(request ReviewListRequest) (ReviewListQuery, error) {
	if request.AlbumID.IsZero() {
		return ReviewListQuery{}, domain.NewValidationError(domain.Violation{Field: "albumId", Description: "is required"})
	}
	pageSize, err := normalizePageSize(request.PageSize)
	if err != nil {
		return ReviewListQuery{}, err
	}
	if request.Order == "" {
		request.Order = ReviewOrderCreatedAt
	}
	if !request.Order.valid() {
		return ReviewListQuery{}, domain.NewValidationError(domain.Violation{Field: "orderBy", Description: "is not supported"})
	}

	query := ReviewListQuery{AlbumID: request.AlbumID, Limit: pageSize + 1, Order: request.Order}
	if request.PageToken == "" {
		return query, nil
	}
	token, err := decodePageToken(request.PageToken)
	if err != nil || token.Resource != reviewPageResource || token.Fingerprint != reviewFingerprint(query) {
		return ReviewListQuery{}, invalidPageToken()
	}
	cursor := ReviewCursor{ID: domain.ID(token.ID)}
	switch query.Order {
	case ReviewOrderCreatedAt:
		if token.CreatedAt == nil {
			return ReviewListQuery{}, invalidPageToken()
		}
		cursor.CreatedAt = token.CreatedAt.UTC()
	case ReviewOrderScore:
		if token.Score == nil || *token.Score < domain.ReviewScoreMin || *token.Score > domain.ReviewScoreMax {
			return ReviewListQuery{}, invalidPageToken()
		}
		cursor.Score = *token.Score
	}
	query.Cursor = &cursor
	return query, nil
}

func reviewPage(query ReviewListQuery, rows []domain.Review) (ReviewPage, error) {
	pageSize := query.Limit - 1
	if len(rows) <= pageSize {
		return ReviewPage{Reviews: rows}, nil
	}
	items := rows[:pageSize]
	last := items[len(items)-1]
	token := pageToken{
		Version:     pageTokenVersion,
		Resource:    reviewPageResource,
		Fingerprint: reviewFingerprint(query),
		ID:          last.ID.String(),
	}
	switch query.Order {
	case ReviewOrderCreatedAt:
		value := last.CreatedAt.UTC()
		token.CreatedAt = &value
	case ReviewOrderScore:
		value := last.Score
		token.Score = &value
	}
	encoded, err := encodePageToken(token)
	if err != nil {
		return ReviewPage{}, fmt.Errorf("encode review page token: %w", err)
	}
	return ReviewPage{Reviews: items, NextPageToken: encoded}, nil
}

func (order ReviewOrder) valid() bool {
	return order == ReviewOrderCreatedAt || order == ReviewOrderScore
}

func reviewFingerprint(query ReviewListQuery) string {
	return fingerprint(query.AlbumID.String(), string(query.Order))
}
