package httpapi

import (
	"context"
	"fmt"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/transport/http/openapi"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

func (s *Server) ListReviews(ctx context.Context, request openapi.ListReviewsRequestObject) (openapi.ListReviewsResponseObject, error) {
	page, err := s.reviews.List(ctx, usecase.ReviewListRequest{
		AlbumID:   domain.ID(request.AlbumId.String()),
		PageSize:  valueOrZero(request.Params.PageSize),
		PageToken: valueOrEmpty(request.Params.PageToken),
		Order:     usecase.ReviewOrder(valueOrEmpty(request.Params.OrderBy)),
	})
	if err != nil {
		return nil, err
	}
	reviews, err := reviewViews(page.Reviews)
	if err != nil {
		return nil, err
	}
	response := openapi.ListReviewsResponse{Reviews: reviews}
	if page.NextPageToken != "" {
		response.NextPageToken = &page.NextPageToken
	}
	return openapi.ListReviews200JSONResponse(response), nil
}

func (s *Server) CreateReview(ctx context.Context, request openapi.CreateReviewRequestObject) (openapi.CreateReviewResponseObject, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, missingBodyField("body")
	}
	review, err := s.reviews.Create(ctx, user.ID, domain.ID(request.AlbumId.String()), domain.ReviewDraft{
		Headline: request.Body.Headline,
		Body:     request.Body.Body,
		Score:    request.Body.Score,
	})
	if err != nil {
		return nil, err
	}
	view, err := reviewView(review)
	if err != nil {
		return nil, err
	}
	location := fmt.Sprintf("/api/v1/albums/%s/reviews/%s", review.AlbumID, review.ID)
	return openapi.CreateReview201JSONResponse{Body: view, Headers: openapi.CreateReview201ResponseHeaders{Location: &location}}, nil
}

func (s *Server) GetReview(ctx context.Context, request openapi.GetReviewRequestObject) (openapi.GetReviewResponseObject, error) {
	review, err := s.reviews.Get(ctx, domain.ID(request.AlbumId.String()), domain.ID(request.ReviewId.String()))
	if err != nil {
		return nil, err
	}
	view, err := reviewView(review)
	if err != nil {
		return nil, err
	}
	return openapi.GetReview200JSONResponse(view), nil
}

func (s *Server) UpdateReview(ctx context.Context, request openapi.UpdateReviewRequestObject) (openapi.UpdateReviewResponseObject, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, missingBodyField("body")
	}
	review, err := s.reviews.Update(
		ctx,
		user.ID,
		domain.ID(request.AlbumId.String()),
		domain.ID(request.ReviewId.String()),
		domain.ReviewPatch{
			Headline: request.Body.Headline,
			Body:     request.Body.Body,
			Score:    request.Body.Score,
			Version:  request.Body.Version,
		},
	)
	if err != nil {
		return nil, err
	}
	view, err := reviewView(review)
	if err != nil {
		return nil, err
	}
	return openapi.UpdateReview200JSONResponse(view), nil
}

func (s *Server) DeleteReview(ctx context.Context, request openapi.DeleteReviewRequestObject) (openapi.DeleteReviewResponseObject, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.reviews.Delete(
		ctx,
		user.ID,
		domain.ID(request.AlbumId.String()),
		domain.ID(request.ReviewId.String()),
		request.Params.Version,
	); err != nil {
		return nil, err
	}
	return openapi.DeleteReview204Response{}, nil
}
