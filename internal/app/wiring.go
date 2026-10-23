package app

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mmmIlia/badgoodtune/internal/clock"
	"github.com/mmmIlia/badgoodtune/internal/config"
	"github.com/mmmIlia/badgoodtune/internal/metrics"
	"github.com/mmmIlia/badgoodtune/internal/repository/postgres"
	"github.com/mmmIlia/badgoodtune/internal/security"
	httpapi "github.com/mmmIlia/badgoodtune/internal/transport/http"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

func buildHandler(
	authConfig config.Auth,
	httpOptions httpapi.Options,
	database *pgxpool.Pool,
	logger *slog.Logger,
	observer *metrics.Metrics,
) (http.Handler, error) {
	store, err := postgres.NewStore(database)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL store: %w", err)
	}
	passwords, err := security.NewArgon2Hasher(security.DefaultArgon2Params(authConfig.Argon2MaxConcurrent))
	if err != nil {
		return nil, fmt.Errorf("create password hasher: %w", err)
	}

	systemClock := clock.System{}
	auth, err := usecase.NewAuthService(store, passwords, security.SessionTokenSource{}, systemClock, authConfig.SessionTTL, observer)
	if err != nil {
		return nil, fmt.Errorf("create authentication service: %w", err)
	}
	albums, err := usecase.NewAlbumService(store, systemClock, observer)
	if err != nil {
		return nil, fmt.Errorf("create album service: %w", err)
	}
	reviews, err := usecase.NewReviewService(store, store, systemClock, observer)
	if err != nil {
		return nil, fmt.Errorf("create review service: %w", err)
	}

	handler, err := httpapi.NewHandler(httpapi.Dependencies{
		Auth:     auth,
		Albums:   albums,
		Reviews:  reviews,
		Probe:    store,
		Logger:   logger,
		Observer: observer,
	}, httpOptions)
	if err != nil {
		return nil, fmt.Errorf("create HTTP handler: %w", err)
	}
	return handler, nil
}
