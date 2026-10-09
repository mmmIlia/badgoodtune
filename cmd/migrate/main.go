package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/mmmIlia/badgoodtune/internal/config"
	"github.com/mmmIlia/badgoodtune/internal/logging"
	"github.com/mmmIlia/badgoodtune/internal/repository/postgres"
)

func main() {
	logger := logging.New(os.Stdout, "info")
	if err := run(); err != nil {
		logger.Error("migration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("migrations are up to date")
}

func run() error {
	settings, err := config.LoadDatabaseConnection()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	connectCtx, cancel := context.WithTimeout(ctx, settings.ConnectTimeout)
	defer cancel()

	database, err := sql.Open("pgx", settings.URL)
	if err != nil {
		return fmt.Errorf("open PostgreSQL: %w", err)
	}
	defer func() { _ = database.Close() }()
	if err := database.PingContext(connectCtx); err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}
	if err := postgres.Migrate(ctx, database); err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		return err
	}
	return nil
}
