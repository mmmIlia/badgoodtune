package httpapi

import (
	"context"
	"fmt"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/transport/http/openapi"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

func (s *Server) ListAlbums(ctx context.Context, request openapi.ListAlbumsRequestObject) (openapi.ListAlbumsResponseObject, error) {
	params := request.Params
	page, err := s.albums.List(ctx, usecase.AlbumListRequest{
		PageSize:    valueOrZero(params.PageSize),
		PageToken:   valueOrEmpty(params.PageToken),
		Search:      valueOrEmpty(params.Q),
		Genre:       valueOrEmpty(params.Genre),
		ReleaseYear: params.ReleaseYear,
		Order:       usecase.AlbumOrder(valueOrEmpty(params.OrderBy)),
	})
	if err != nil {
		return nil, err
	}
	albums, err := albumViews(page.Albums)
	if err != nil {
		return nil, err
	}
	response := openapi.ListAlbumsResponse{Albums: albums}
	if page.NextPageToken != "" {
		response.NextPageToken = &page.NextPageToken
	}
	return openapi.ListAlbums200JSONResponse(response), nil
}

func (s *Server) CreateAlbum(ctx context.Context, request openapi.CreateAlbumRequestObject) (openapi.CreateAlbumResponseObject, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, missingBodyField("body")
	}
	album, err := s.albums.Create(ctx, user.ID, domain.AlbumDraft{
		Artist:      request.Body.Artist,
		Title:       request.Body.Title,
		ReleaseYear: request.Body.ReleaseYear,
		Genre:       request.Body.Genre,
	})
	if err != nil {
		return nil, err
	}
	view, err := albumView(album)
	if err != nil {
		return nil, err
	}
	location := fmt.Sprintf("/api/v1/albums/%s", album.ID)
	return openapi.CreateAlbum201JSONResponse{Body: view, Headers: openapi.CreateAlbum201ResponseHeaders{Location: &location}}, nil
}

func (s *Server) GetAlbum(ctx context.Context, request openapi.GetAlbumRequestObject) (openapi.GetAlbumResponseObject, error) {
	album, err := s.albums.Get(ctx, domain.ID(request.AlbumId.String()))
	if err != nil {
		return nil, err
	}
	view, err := albumView(album)
	if err != nil {
		return nil, err
	}
	return openapi.GetAlbum200JSONResponse(view), nil
}

func (s *Server) UpdateAlbum(ctx context.Context, request openapi.UpdateAlbumRequestObject) (openapi.UpdateAlbumResponseObject, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, missingBodyField("body")
	}
	album, err := s.albums.Update(ctx, user.ID, domain.ID(request.AlbumId.String()), domain.AlbumPatch{
		Artist:      request.Body.Artist,
		Title:       request.Body.Title,
		ReleaseYear: request.Body.ReleaseYear,
		Genre:       request.Body.Genre,
		Version:     request.Body.Version,
	})
	if err != nil {
		return nil, err
	}
	view, err := albumView(album)
	if err != nil {
		return nil, err
	}
	return openapi.UpdateAlbum200JSONResponse(view), nil
}

func (s *Server) DeleteAlbum(ctx context.Context, request openapi.DeleteAlbumRequestObject) (openapi.DeleteAlbumResponseObject, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.albums.Delete(ctx, user.ID, domain.ID(request.AlbumId.String()), request.Params.Version); err != nil {
		return nil, err
	}
	return openapi.DeleteAlbum204Response{}, nil
}
