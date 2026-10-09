package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/mmmIlia/badgoodtune/internal/transport/http/openapi"
)

const (
	maxRequestIDLength      = 64
	requestIDEntropyBytes   = 16
	requestIDFallbackLayout = "20060102T150405.000000000Z"
)

type requestIDContextKey struct{}

type responseRecorder struct {
	http.ResponseWriter
	status    int
	bytes     int
	operation string
}

func (w *responseRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *responseRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseRecorder) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	written, err := w.ResponseWriter.Write(body)
	w.bytes += written
	return written, err
}

func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if !validRequestID(requestID) {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		level := slog.LevelInfo
		if strings.HasPrefix(r.URL.Path, "/health/") && status < http.StatusBadRequest {
			level = slog.LevelDebug
		}
		s.logger.LogAttrs(r.Context(), level, "HTTP request",
			slog.String("request_id", requestIDFromContext(r.Context())),
			slog.String("method", r.Method),
			slog.String("operation", requestOperation(recorder, r)),
			slog.Int("status", status),
			slog.Int("response_bytes", recorder.bytes),
			slog.Duration("duration", time.Since(started)),
		)
	})
}

func (s *Server) recordOperation(next openapi.StrictHandlerFunc, operationID string) openapi.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		if recorder, ok := w.(*responseRecorder); ok {
			recorder.operation = operationID
		}
		return next(ctx, w, r, request)
	}
}

func requestOperation(w http.ResponseWriter, r *http.Request) string {
	if recorder, ok := w.(*responseRecorder); ok && recorder.operation != "" {
		return recorder.operation
	}
	if r.Pattern != "" {
		return r.Pattern
	}
	return "unmatched"
}

func (s *Server) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.ErrorContext(r.Context(), "panic recovered",
					slog.String("request_id", requestIDFromContext(r.Context())),
					slog.Any("panic", recovered),
					slog.String("stack", string(debug.Stack())),
				)
				if recorder, ok := w.(*responseRecorder); ok && recorder.status != 0 {
					return
				}
				s.writeProblemDefinition(w, r, problemForStatus(http.StatusInternalServerError), nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()
		headers.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		headers.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		headers.Set("Referrer-Policy", "no-referrer")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) limitRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if methodHasJSONBody(r.Method) && !isJSONContentType(r.Header.Get("Content-Type")) {
			s.writeProblemDefinition(w, r, problemForStatus(http.StatusUnsupportedMediaType), nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func methodHasJSONBody(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch
}

func isJSONContentType(value string) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func validRequestID(value string) bool {
	if value == "" || len(value) > maxRequestIDLength {
		return false
	}
	for _, character := range value {
		if !(character >= 'a' && character <= 'z') &&
			!(character >= 'A' && character <= 'Z') &&
			!(character >= '0' && character <= '9') &&
			character != '-' && character != '_' {
			return false
		}
	}
	return true
}

func newRequestID() string {
	random := make([]byte, requestIDEntropyBytes)
	if _, err := rand.Read(random); err == nil {
		return base64.RawURLEncoding.EncodeToString(random)
	}
	return strings.ReplaceAll(time.Now().UTC().Format(requestIDFallbackLayout), ".", "-")
}
