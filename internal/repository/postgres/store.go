package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mmmIlia/badgoodtune/internal/repository/postgres/sqlc"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

var (
	_ usecase.AuthRepository   = (*Store)(nil)
	_ usecase.AlbumRepository  = (*Store)(nil)
	_ usecase.ReviewRepository = (*Store)(nil)
)

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewStore(pool *pgxpool.Pool) (*Store, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL pool is required")
	}
	return &Store{pool: pool, queries: sqlc.New(pool)}, nil
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
