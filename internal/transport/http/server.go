package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/mmmIlia/badgoodtune/internal/transport/http/openapi"
)

const maxRequestBodyBytes int64 = 1 << 20

type Server struct {
	auth             AuthUseCases
	albums           AlbumUseCases
	reviews          ReviewUseCases
	probe            ReadinessProbe
	logger           *slog.Logger
	observer         HTTPObserver
	cookieName       string
	cookieSecure     bool
	readinessTimeout time.Duration
}

func NewHandler(deps Dependencies, options Options) (http.Handler, error) {
	if deps.Auth == nil || deps.Albums == nil || deps.Reviews == nil || deps.Probe == nil {
		return nil, errors.New("HTTP API dependencies must not be nil")
	}
	if deps.Logger == nil {
		return nil, errors.New("HTTP API logger is required")
	}
	if deps.Observer == nil {
		return nil, errors.New("HTTP observer is required")
	}
	if options.CookieName == "" {
		return nil, errors.New("HTTP API cookie name is required")
	}
	if options.ReadinessTimeout <= 0 {
		return nil, errors.New("HTTP API readiness timeout must be positive")
	}

	server := &Server{
		auth:             deps.Auth,
		albums:           deps.Albums,
		reviews:          deps.Reviews,
		probe:            deps.Probe,
		logger:           deps.Logger,
		observer:         deps.Observer,
		cookieName:       options.CookieName,
		cookieSecure:     options.CookieSecure,
		readinessTimeout: options.ReadinessTimeout,
	}
	api, health, err := server.contractHandlers()
	if err != nil {
		return nil, err
	}

	routes := http.NewServeMux()
	routes.Handle("/api/", api)
	routes.Handle("/health/", health)
	return server.middleware(routes), nil
}

func (s *Server) middleware(next http.Handler) http.Handler {
	crossOriginProtection := http.NewCrossOriginProtection()
	crossOriginProtection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		definition := problemForStatus(http.StatusForbidden)
		definition.detail = "Cross-origin state-changing requests are not allowed."
		s.writeProblemDefinition(w, r, definition, nil)
	}))

	next = crossOriginProtection.Handler(next)
	next = s.securityHeaders(next)
	next = s.recoverPanics(next)
	next = s.accessLog(next)
	return s.requestID(next)
}

var _ openapi.StrictServerInterface = (*Server)(nil)
