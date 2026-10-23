package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	App      App
	HTTP     HTTP
	Metrics  Metrics
	Database Database
	Auth     Auth
	Logging  Logging
}

type App struct {
	Environment string `env:"APP_ENV" env-default:"development"`
}

type HTTP struct {
	Address           string        `env:"HTTP_ADDRESS" env-default:":8080"`
	ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"10s"`
	ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" env-default:"5s"`
	ReadTimeout       time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout      time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"30s"`
	IdleTimeout       time.Duration `env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ReadinessTimeout  time.Duration `env:"READINESS_TIMEOUT" env-default:"1s"`
}

type Database struct {
	Connection DatabaseConnection
	Pool       DatabasePool
	Queries    DatabaseQueries
}

type DatabaseQueries struct {
	StatementTimeout time.Duration `env:"DATABASE_STATEMENT_TIMEOUT" env-default:"5s"`
}

type DatabaseConnection struct {
	URL            string        `env:"DATABASE_URL" env-required:"true"`
	ConnectTimeout time.Duration `env:"DATABASE_CONNECT_TIMEOUT" env-default:"5s"`
}

type DatabasePool struct {
	MaxConnections int32 `env:"DATABASE_MAX_CONNS" env-default:"10"`
	MinConnections int32 `env:"DATABASE_MIN_CONNS" env-default:"1"`
}

type Auth struct {
	CookieName          string        `env:"COOKIE_NAME" env-default:"badgoodtune_session"`
	CookieSecure        bool          `env:"COOKIE_SECURE" env-default:"false"`
	SessionTTL          time.Duration `env:"SESSION_TTL" env-default:"168h"`
	Argon2MaxConcurrent int           `env:"ARGON2_MAX_CONCURRENT" env-default:"2"`
}

type Logging struct {
	Level string `env:"LOG_LEVEL" env-default:"info"`
}

func Load() (Config, error) {
	var settings Config
	if err := cleanenv.ReadEnv(&settings); err != nil {
		return Config{}, fmt.Errorf("read environment: %w", err)
	}
	if err := settings.Validate(); err != nil {
		return Config{}, err
	}
	return settings, nil
}

func LoadDatabaseConnection() (DatabaseConnection, error) {
	var settings DatabaseConnection
	if err := cleanenv.ReadEnv(&settings); err != nil {
		return DatabaseConnection{}, fmt.Errorf("read database environment: %w", err)
	}
	if err := settings.Validate(); err != nil {
		return DatabaseConnection{}, err
	}
	return settings, nil
}
