package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

type reviewQueryRow struct {
	ID              pgtype.UUID
	AlbumID         pgtype.UUID
	AuthorID        pgtype.UUID
	AuthorUsername  string
	AuthorCreatedAt pgtype.Timestamptz
	Headline        string
	Body            string
	Score           int16
	Version         int64
	CreatedAt       pgtype.Timestamptz
	UpdatedAt       pgtype.Timestamptz
}

func reviewFromQueryRow(row reviewQueryRow) (domain.Review, error) {
	id, err := idFromPG(row.ID)
	if err != nil {
		return domain.Review{}, fmt.Errorf("map review id: %w", err)
	}
	albumID, err := idFromPG(row.AlbumID)
	if err != nil {
		return domain.Review{}, fmt.Errorf("map review album id: %w", err)
	}
	author, err := userFromRecord(row.AuthorID, row.AuthorUsername, row.AuthorCreatedAt)
	if err != nil {
		return domain.Review{}, fmt.Errorf("map review author: %w", err)
	}
	createdAt, err := timeFromPG(row.CreatedAt)
	if err != nil {
		return domain.Review{}, fmt.Errorf("map review creation time: %w", err)
	}
	updatedAt, err := timeFromPG(row.UpdatedAt)
	if err != nil {
		return domain.Review{}, fmt.Errorf("map review update time: %w", err)
	}
	return domain.Review{
		ID:        id,
		AlbumID:   albumID,
		Author:    author,
		Headline:  row.Headline,
		Body:      row.Body,
		Score:     int(row.Score),
		Version:   row.Version,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func reviewsFromQueryRows[T any](rows []T, convert func(T) reviewQueryRow) ([]domain.Review, error) {
	reviews := make([]domain.Review, 0, len(rows))
	for _, row := range rows {
		review, err := reviewFromQueryRow(convert(row))
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}
	return reviews, nil
}
