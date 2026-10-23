package app

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mmmIlia/badgoodtune/internal/config"
)

func connectDatabase(ctx context.Context, settings config.Database) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(settings.Connection.URL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	poolConfig.MaxConns = settings.Pool.MaxConnections
	poolConfig.MinConns = settings.Pool.MinConnections
	poolConfig.ConnConfig.ConnectTimeout = settings.Connection.ConnectTimeout
	poolConfig.ConnConfig.RuntimeParams["application_name"] = "badgoodtune-server"
	poolConfig.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(settings.Queries.StatementTimeout.Milliseconds(), 10)

	connectContext, cancel := context.WithTimeout(ctx, settings.Connection.ConnectTimeout)
	defer cancel()
	database, err := pgxpool.NewWithConfig(connectContext, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	if err := database.Ping(connectContext); err != nil {
		database.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	return database, nil
}
