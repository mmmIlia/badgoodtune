package httpapi

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/transport/http/openapi"
)

func albumView(album domain.Album) (openapi.Album, error) {
	id, err := uuid.Parse(album.ID.String())
	if err != nil {
		return openapi.Album{}, fmt.Errorf("map album ID: %w", err)
	}
	owner, err := userView(album.Owner)
	if err != nil {
		return openapi.Album{}, fmt.Errorf("map album owner: %w", err)
	}
	return openapi.Album{
		Id:           id,
		Owner:        owner,
		Artist:       album.Artist,
		Title:        album.Title,
		ReleaseYear:  album.ReleaseYear,
		Genre:        album.Genre,
		AverageScore: album.AverageScore,
		ReviewCount:  album.ReviewCount,
		Version:      album.Version,
		CreatedAt:    album.CreatedAt.UTC(),
		UpdatedAt:    album.UpdatedAt.UTC(),
	}, nil
}

func albumViews(albums []domain.Album) ([]openapi.Album, error) {
	views := make([]openapi.Album, 0, len(albums))
	for _, album := range albums {
		view, err := albumView(album)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}
