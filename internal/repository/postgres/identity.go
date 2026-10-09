package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/repository/postgres/sqlc"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

func (s *Store) CreateUserWithSession(
	ctx context.Context,
	username string,
	passwordHash string,
	session usecase.NewSession,
) (domain.User, error) {
	var row sqlc.CreateUserRow
	err := s.inTransaction(ctx, func(queries *sqlc.Queries) error {
		if err := deleteExpiredSessions(ctx, queries, session.CreatedAt); err != nil {
			return err
		}
		created, err := queries.CreateUser(ctx, sqlc.CreateUserParams{
			Username:     username,
			PasswordHash: passwordHash,
			CreatedAt:    timeToPG(session.CreatedAt),
		})
		if err != nil {
			return mapWriteError(err)
		}
		row = created
		if err := queries.CreateSession(ctx, sqlc.CreateSessionParams{
			TokenHash: session.TokenHash,
			UserID:    row.ID,
			ExpiresAt: timeToPG(session.ExpiresAt),
			CreatedAt: timeToPG(session.CreatedAt),
		}); err != nil {
			return mapWriteError(err)
		}
		return nil
	})
	if err != nil {
		return domain.User{}, err
	}

	user, err := userFromRecord(row.ID, row.Username, row.CreatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("map created user: %w", err)
	}
	return user, nil
}

func (s *Store) GetCredentialByUsername(ctx context.Context, username string) (usecase.Credential, error) {
	row, err := s.queries.GetCredentialByUsername(ctx, username)
	if err != nil {
		return usecase.Credential{}, mapNotFound(err)
	}
	user, err := userFromRecord(row.ID, row.Username, row.CreatedAt)
	if err != nil {
		return usecase.Credential{}, fmt.Errorf("map credential user: %w", err)
	}
	return usecase.Credential{
		User:         user,
		PasswordHash: row.PasswordHash,
	}, nil
}

func (s *Store) CreateSession(ctx context.Context, userID domain.ID, session usecase.NewSession) error {
	userUUID, err := idToPG(userID)
	if err != nil {
		return err
	}
	return s.inTransaction(ctx, func(queries *sqlc.Queries) error {
		if err := deleteExpiredSessions(ctx, queries, session.CreatedAt); err != nil {
			return err
		}
		if err := queries.CreateSession(ctx, sqlc.CreateSessionParams{
			TokenHash: session.TokenHash,
			UserID:    userUUID,
			ExpiresAt: timeToPG(session.ExpiresAt),
			CreatedAt: timeToPG(session.CreatedAt),
		}); err != nil {
			return mapWriteError(err)
		}
		return nil
	})
}

func deleteExpiredSessions(ctx context.Context, queries *sqlc.Queries, now time.Time) error {
	if _, err := queries.DeleteExpiredSessions(ctx, timeToPG(now)); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}

func (s *Store) GetUserBySession(ctx context.Context, tokenHash []byte, now time.Time) (domain.User, error) {
	row, err := s.queries.GetUserBySession(ctx, sqlc.GetUserBySessionParams{
		TokenHash: tokenHash,
		ExpiresAt: timeToPG(now),
	})
	if err != nil {
		return domain.User{}, mapNotFound(err)
	}
	user, err := userFromRecord(row.ID, row.Username, row.CreatedAt)
	if err != nil {
		return domain.User{}, fmt.Errorf("map authenticated user: %w", err)
	}
	return user, nil
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	return s.queries.DeleteSession(ctx, tokenHash)
}
