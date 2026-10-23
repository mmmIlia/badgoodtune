package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mmmIlia/badgoodtune/internal/config"
	"github.com/mmmIlia/badgoodtune/internal/metrics"
	httpapi "github.com/mmmIlia/badgoodtune/internal/transport/http"
)

const maxHeaderBytes = 1 << 20

type App struct {
	server          *http.Server
	metricsServer   *http.Server
	database        *pgxpool.Pool
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

func New(
	ctx context.Context,
	databaseConfig config.Database,
	authConfig config.Auth,
	httpConfig config.HTTP,
	metricsConfig config.Metrics,
	logger *slog.Logger,
) (*App, error) {
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	database, err := connectDatabase(ctx, databaseConfig)
	if err != nil {
		return nil, err
	}

	observations := metrics.New()
	handler, err := buildHandler(authConfig, httpapi.Options{
		CookieName:       authConfig.CookieName,
		CookieSecure:     authConfig.CookieSecure,
		ReadinessTimeout: httpConfig.ReadinessTimeout,
	}, database, logger, observations)
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
		server: server,
		metricsServer: &http.Server{
			Addr:              metricsConfig.Address,
			Handler:           observations.Handler(),
			ReadHeaderTimeout: metricsConfig.ReadHeaderTimeout,
			ReadTimeout:       metricsConfig.ReadHeaderTimeout,
			WriteTimeout:      metricsConfig.WriteTimeout,
			IdleTimeout:       metricsConfig.IdleTimeout,
			MaxHeaderBytes:    maxHeaderBytes,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
		},
		database:        database,
		logger:          logger,
		shutdownTimeout: httpConfig.ShutdownTimeout,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.database.Close()

	a.logger.Info("HTTP servers starting", slog.String("api", a.server.Addr), slog.String("metrics", a.metricsServer.Addr))
	return runHTTPServers(ctx, a.shutdownTimeout, a.server, a.metricsServer)
}
