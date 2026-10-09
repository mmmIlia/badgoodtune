package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

type albumQueryRow struct {
	ID             pgtype.UUID
	OwnerID        pgtype.UUID
	OwnerUsername  string
	OwnerCreatedAt pgtype.Timestamptz
	Artist         string
	Title          string
	ReleaseYear    int16
	Genre          string
	AverageScore   float64
	ReviewCount    int64
	Version        int64
	CreatedAt      pgtype.Timestamptz
	UpdatedAt      pgtype.Timestamptz
}

func albumFromQueryRow(row albumQueryRow) (domain.Album, error) {
	id, err := idFromPG(row.ID)
	if err != nil {
		return domain.Album{}, fmt.Errorf("map album id: %w", err)
	}
	owner, err := userFromRecord(row.OwnerID, row.OwnerUsername, row.OwnerCreatedAt)
	if err != nil {
		return domain.Album{}, fmt.Errorf("map album owner: %w", err)
	}
	createdAt, err := timeFromPG(row.CreatedAt)
	if err != nil {
		return domain.Album{}, fmt.Errorf("map album creation time: %w", err)
	}
	updatedAt, err := timeFromPG(row.UpdatedAt)
	if err != nil {
		return domain.Album{}, fmt.Errorf("map album update time: %w", err)
	}

	var averageScore *float64
	if row.ReviewCount > 0 {
		value := row.AverageScore
		averageScore = &value
	}
	return domain.Album{
		ID:           id,
		Owner:        owner,
		Artist:       row.Artist,
		Title:        row.Title,
		ReleaseYear:  int(row.ReleaseYear),
		Genre:        row.Genre,
		AverageScore: averageScore,
		ReviewCount:  row.ReviewCount,
		Version:      row.Version,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

func albumsFromQueryRows[T any](rows []T, convert func(T) albumQueryRow) ([]domain.Album, error) {
	albums := make([]domain.Album, 0, len(rows))
	for _, row := range rows {
		album, err := albumFromQueryRow(convert(row))
		if err != nil {
			return nil, err
		}
		albums = append(albums, album)
	}
	return albums, nil
}

func optionalYear(year *int) *int16 {
	if year == nil {
		return nil
	}
	value := int16(*year)
	return &value
}
