package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mmmIlia/badgoodtune/internal/config"
)

func TestLoadPopulatesNestedSections(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("HTTP_ADDRESS", ":9090")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("COOKIE_NAME", "test_session")
	t.Setenv("LOG_LEVEL", "debug")

	settings, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "test", settings.App.Environment)
	assert.Equal(t, ":9090", settings.HTTP.Address)
	assert.Equal(t, "postgres://example", settings.Database.Connection.URL)
	assert.Equal(t, "test_session", settings.Auth.CookieName)
	assert.Equal(t, "debug", settings.Logging.Level)
}

func TestLoadDatabaseConnectionIgnoresServerPoolSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "2s")
	t.Setenv("DATABASE_MAX_CONNS", "0")

	settings, err := config.LoadDatabaseConnection()
	require.NoError(t, err)
	assert.Equal(t, "postgres://example", settings.URL)
	assert.Equal(t, 2*time.Second, settings.ConnectTimeout)
}

func TestProductionConfigRequiresSecureHostCookie(t *testing.T) {
	t.Parallel()

	settings := config.Config{
		App: config.App{Environment: "production"},
		HTTP: config.HTTP{
			Address: ":8080", ShutdownTimeout: time.Second,
			ReadHeaderTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second,
			IdleTimeout: time.Second, ReadinessTimeout: time.Second,
		},
		Database: config.Database{
			Connection: config.DatabaseConnection{URL: "postgres://example", ConnectTimeout: time.Second},
			Pool:       config.DatabasePool{MaxConnections: 2, MinConnections: 1},
		},
		Auth: config.Auth{
			CookieName: "badgoodtune_session", SessionTTL: 24 * time.Hour,
			Argon2MaxConcurrent: 1,
		},
		Logging: config.Logging{Level: "info"},
	}

	err := settings.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "COOKIE_SECURE")
	assert.ErrorContains(t, err, "__Host-")
}
