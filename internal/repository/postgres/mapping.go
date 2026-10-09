package postgres

import (
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

func idToPG(id domain.ID) (pgtype.UUID, error) {
	var value pgtype.UUID
	if err := value.Scan(id.String()); err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse UUID %q: %w", id, err)
	}
	if !value.Valid {
		return pgtype.UUID{}, errors.New("UUID is null")
	}
	return value, nil
}

func idFromPG(id pgtype.UUID) (domain.ID, error) {
	if !id.Valid {
		return "", errors.New("UUID is null")
	}
	return domain.ID(id.String()), nil
}

func timeToPG(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func timeFromPG(value pgtype.Timestamptz) (time.Time, error) {
	if !value.Valid || value.InfinityModifier != pgtype.Finite {
		return time.Time{}, errors.New("timestamp is null or infinite")
	}
	return value.Time.UTC(), nil
}

func userFromRecord(id pgtype.UUID, username string, createdAt pgtype.Timestamptz) (domain.User, error) {
	userID, err := idFromPG(id)
	if err != nil {
		return domain.User{}, fmt.Errorf("map user id: %w", err)
	}
	userCreatedAt, err := timeFromPG(createdAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("map user creation time: %w", err)
	}
	return domain.User{ID: userID, Username: username, CreatedAt: userCreatedAt}, nil
}
