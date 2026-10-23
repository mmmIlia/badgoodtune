package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	minDatabaseConnections int32 = 1
	maxDatabaseConnections int32 = 100
	minIdleConnections     int32 = 0
	minArgon2Concurrency         = 1
	maxArgon2Concurrency         = 64
	minSessionTTL                = 5 * time.Minute
	maxSessionTTL                = 30 * 24 * time.Hour
)

func (c Config) Validate() error {
	problems := c.App.problems()
	problems = append(problems, c.HTTP.problems()...)
	problems = append(problems, c.Metrics.problems()...)
	if c.HTTP.Address == c.Metrics.Address {
		problems = append(problems, "HTTP_ADDRESS and METRICS_ADDRESS must differ")
	}
	problems = append(problems, c.Database.problems()...)
	problems = append(problems, c.Auth.problems(c.App.Environment)...)
	problems = append(problems, c.Logging.problems()...)

	if len(problems) == 0 {
		return nil
	}
	return errors.New("invalid configuration: " + strings.Join(problems, "; "))
}

func (c DatabaseConnection) Validate() error {
	problems := c.problems()
	if len(problems) == 0 {
		return nil
	}
	return errors.New("invalid database configuration: " + strings.Join(problems, "; "))
}

func (a App) problems() []string {
	switch a.Environment {
	case "development", "test", "production":
		return nil
	default:
		return []string{"APP_ENV must be development, test, or production"}
	}
}

func (h HTTP) problems() []string {
	var problems []string
	if strings.TrimSpace(h.Address) == "" {
		problems = append(problems, "HTTP_ADDRESS must not be empty")
	}
	if h.ShutdownTimeout <= 0 {
		problems = append(problems, "SHUTDOWN_TIMEOUT must be positive")
	}
	if h.ReadHeaderTimeout <= 0 {
		problems = append(problems, "HTTP_READ_HEADER_TIMEOUT must be positive")
	}
	if h.ReadTimeout <= 0 {
		problems = append(problems, "HTTP_READ_TIMEOUT must be positive")
	}
	if h.WriteTimeout <= 0 {
		problems = append(problems, "HTTP_WRITE_TIMEOUT must be positive")
	}
	if h.IdleTimeout <= 0 {
		problems = append(problems, "HTTP_IDLE_TIMEOUT must be positive")
	}
	if h.ReadinessTimeout <= 0 {
		problems = append(problems, "READINESS_TIMEOUT must be positive")
	}
	return problems
}

func (d Database) problems() []string {
	problems := d.Connection.problems()
	return append(problems, d.Pool.problems()...)
}

func (c DatabaseConnection) problems() []string {
	var problems []string
	if strings.TrimSpace(c.URL) == "" {
		problems = append(problems, "DATABASE_URL must not be empty")
	}
	if c.ConnectTimeout <= 0 {
		problems = append(problems, "DATABASE_CONNECT_TIMEOUT must be positive")
	}
	return problems
}

func (p DatabasePool) problems() []string {
	var problems []string
	if p.MaxConnections < minDatabaseConnections || p.MaxConnections > maxDatabaseConnections {
		problems = append(problems, fmt.Sprintf(
			"DATABASE_MAX_CONNS must be between %d and %d",
			minDatabaseConnections,
			maxDatabaseConnections,
		))
	}
	if p.MinConnections < minIdleConnections || p.MinConnections > p.MaxConnections {
		problems = append(problems, fmt.Sprintf(
			"DATABASE_MIN_CONNS must be between %d and DATABASE_MAX_CONNS",
			minIdleConnections,
		))
	}
	return problems
}

func (a Auth) problems(environment string) []string {
	var problems []string
	if strings.TrimSpace(a.CookieName) == "" || strings.ContainsAny(a.CookieName, "\r\n\t ;,") {
		problems = append(problems, "COOKIE_NAME is invalid")
	}
	if environment == "production" {
		if !a.CookieSecure {
			problems = append(problems, "COOKIE_SECURE must be true in production")
		}
		if !strings.HasPrefix(a.CookieName, "__Host-") {
			problems = append(problems, "COOKIE_NAME must use the __Host- prefix in production")
		}
	}
	if a.SessionTTL < minSessionTTL || a.SessionTTL > maxSessionTTL {
		problems = append(problems, fmt.Sprintf("SESSION_TTL must be between %s and %s", minSessionTTL, maxSessionTTL))
	}
	if a.Argon2MaxConcurrent < minArgon2Concurrency || a.Argon2MaxConcurrent > maxArgon2Concurrency {
		problems = append(problems, fmt.Sprintf(
			"ARGON2_MAX_CONCURRENT must be between %d and %d",
			minArgon2Concurrency,
			maxArgon2Concurrency,
		))
	}
	return problems
}

func (l Logging) problems() []string {
	switch strings.ToLower(l.Level) {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return []string{"LOG_LEVEL must be debug, info, warn, or error"}
	}
}
