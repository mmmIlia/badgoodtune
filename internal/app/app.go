package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mmmIlia/badgoodtune/internal/config"
	httpapi "github.com/mmmIlia/badgoodtune/internal/transport/http"
)

const maxHeaderBytes = 1 << 20

type App struct {
	server          *http.Server
	database        *pgxpool.Pool
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

func New(
	ctx context.Context,
	databaseConfig config.Database,
	authConfig config.Auth,
	httpConfig config.HTTP,
	logger *slog.Logger,
) (*App, error) {
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	database, err := connectDatabase(ctx, databaseConfig)
	if err != nil {
		return nil, err
	}

	handler, err := buildHandler(authConfig, httpapi.Options{
		CookieName:       authConfig.CookieName,
		CookieSecure:     authConfig.CookieSecure,
		ReadinessTimeout: httpConfig.ReadinessTimeout,
		StaticDir:        httpConfig.StaticDir,
	}, database, logger)
	if err != nil {
		database.Close()
		return nil, err
	}

	server := &http.Server{
		Addr:              httpConfig.Address,
		Handler:           handler,
		ReadHeaderTimeout: httpConfig.ReadHeaderTimeout,
		ReadTimeout:       httpConfig.ReadTimeout,
		WriteTimeout:      httpConfig.WriteTimeout,
		IdleTimeout:       httpConfig.IdleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	return &App{
		server:          server,
		database:        database,
		logger:          logger,
		shutdownTimeout: httpConfig.ShutdownTimeout,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.database.Close()

	listenError := make(chan error, 1)
	go func() {
		a.logger.Info("HTTP server starting", slog.String("address", a.server.Addr))
		listenError <- a.server.ListenAndServe()
	}()

	select {
	case err := <-listenError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		a.logger.Info("shutdown requested")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancel()
	if err := a.server.Shutdown(shutdownContext); err != nil {
		_ = a.server.Close()
		return fmt.Errorf("graceful HTTP shutdown: %w", err)
	}
	if err := <-listenError; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP during shutdown: %w", err)
	}
	a.logger.Info("HTTP server stopped")
	return nil
}
