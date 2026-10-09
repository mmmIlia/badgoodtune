package httpapi

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/transport/http/openapi"
)

func reviewView(review domain.Review) (openapi.Review, error) {
	id, err := uuid.Parse(review.ID.String())
	if err != nil {
		return openapi.Review{}, fmt.Errorf("map review ID: %w", err)
	}
	albumID, err := uuid.Parse(review.AlbumID.String())
	if err != nil {
		return openapi.Review{}, fmt.Errorf("map review album ID: %w", err)
	}
	author, err := userView(review.Author)
	if err != nil {
		return openapi.Review{}, fmt.Errorf("map review author: %w", err)
	}
	return openapi.Review{
		Id:        id,
		AlbumId:   albumID,
		Author:    author,
		Headline:  review.Headline,
		Body:      review.Body,
		Score:     review.Score,
		Version:   review.Version,
		CreatedAt: review.CreatedAt.UTC(),
		UpdatedAt: review.UpdatedAt.UTC(),
	}, nil
}

func reviewViews(reviews []domain.Review) ([]openapi.Review, error) {
	views := make([]openapi.Review, 0, len(reviews))
	for _, review := range reviews {
		view, err := reviewView(review)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}
