package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/transport/http/openapi"
)

type requestDecodeError struct{ cause error }

func (e requestDecodeError) Error() string { return "decode request: " + e.cause.Error() }
func (e requestDecodeError) Unwrap() error { return e.cause }

type unavailableError struct{ cause error }

func (e unavailableError) Error() string { return "dependency unavailable: " + e.cause.Error() }
func (e unavailableError) Unwrap() error { return e.cause }

type problemDefinition struct {
	status int
	code   openapi.ProblemCode
	title  string
	detail string
}

func (s *Server) writeContractProblem(w http.ResponseWriter, r *http.Request, err error, suggestedStatus int) {
	status := suggestedStatus
	if status < 400 || status > 599 {
		status = http.StatusBadRequest
	}
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		status = http.StatusRequestEntityTooLarge
	}
	definition := problemForStatus(status)
	if status == http.StatusBadRequest {
		definition.detail = "The request does not match the API contract."
	}
	s.writeProblemDefinition(w, r, definition, nil)
}

func (s *Server) writeProblem(w http.ResponseWriter, r *http.Request, err error) {
	definition, violations := classifyProblem(err)
	if definition.status >= http.StatusInternalServerError {
		s.logger.ErrorContext(r.Context(), "request failed",
			slog.String("request_id", requestIDFromContext(r.Context())),
			slog.String("method", r.Method),
			slog.String("operation", requestOperation(w, r)),
			slog.String("error", err.Error()),
		)
	}
	s.writeProblemDefinition(w, r, definition, violations)
}

func (s *Server) writeProblemDefinition(w http.ResponseWriter, r *http.Request, definition problemDefinition, violations []openapi.FieldViolation) {
	instance := r.URL.Path
	detail := definition.detail
	problem := openapi.Problem{
		Type:      "about:blank",
		Title:     definition.title,
		Status:    definition.status,
		Code:      definition.code,
		RequestId: requestIDFromContext(r.Context()),
		Instance:  &instance,
	}
	if detail != "" {
		problem.Detail = &detail
	}
	if len(violations) > 0 {
		problem.Violations = &violations
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(definition.status)
	if err := json.NewEncoder(w).Encode(problem); err != nil {
		s.logger.ErrorContext(r.Context(), "encode problem response", slog.String("error", err.Error()))
	}
}

func classifyProblem(err error) (problemDefinition, []openapi.FieldViolation) {
	var validationError *domain.ValidationError
	if errors.As(err, &validationError) {
		violations := make([]openapi.FieldViolation, 0, len(validationError.Violations))
		for _, violation := range validationError.Violations {
			violations = append(violations, openapi.FieldViolation{Field: violation.Field, Description: violation.Description})
		}
		return problemForStatus(http.StatusBadRequest), violations
	}
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		return problemForStatus(http.StatusRequestEntityTooLarge), nil
	}
	var unavailable unavailableError
	if errors.As(err, &unavailable) {
		return problemForStatus(http.StatusServiceUnavailable), nil
	}
	var decodeError requestDecodeError
	if errors.As(err, &decodeError) {
		return problemForStatus(http.StatusBadRequest), nil
	}
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrUnauthenticated):
		return problemForStatus(http.StatusUnauthorized), nil
	case errors.Is(err, domain.ErrPermissionDenied):
		return problemForStatus(http.StatusForbidden), nil
	case errors.Is(err, domain.ErrNotFound):
		return problemForStatus(http.StatusNotFound), nil
	case errors.Is(err, domain.ErrAlbumHasReviews):
		definition := problemForStatus(http.StatusConflict)
		definition.detail = "Delete the album's reviews before deleting the album."
		return definition, nil
	case errors.Is(err, domain.ErrConflict):
		return problemForStatus(http.StatusConflict), nil
	case errors.Is(err, domain.ErrPreconditionFailed):
		return problemForStatus(http.StatusPreconditionFailed), nil
	default:
		return problemForStatus(http.StatusInternalServerError), nil
	}
}

func problemForStatus(status int) problemDefinition {
	switch status {
	case http.StatusBadRequest:
		return problemDefinition{status: status, code: openapi.INVALIDARGUMENT, title: "Invalid request", detail: "One or more request fields are invalid."}
	case http.StatusUnauthorized:
		return problemDefinition{status: status, code: openapi.UNAUTHENTICATED, title: "Authentication required", detail: "The credentials are missing, invalid, or expired."}
	case http.StatusForbidden:
		return problemDefinition{status: status, code: openapi.PERMISSIONDENIED, title: "Permission denied", detail: "You do not own this resource."}
	case http.StatusNotFound:
		return problemDefinition{status: status, code: openapi.NOTFOUND, title: "Resource not found", detail: "The requested resource does not exist."}
	case http.StatusConflict:
		return problemDefinition{status: status, code: openapi.CONFLICT, title: "Resource conflict", detail: "The resource conflicts with existing data."}
	case http.StatusPreconditionFailed:
		return problemDefinition{status: status, code: openapi.FAILEDPRECONDITION, title: "Resource changed", detail: "Refresh the resource and retry with its current version."}
	case http.StatusRequestEntityTooLarge:
		return problemDefinition{status: status, code: openapi.PAYLOADTOOLARGE, title: "Payload too large", detail: "The request body exceeds the allowed size."}
	case http.StatusUnsupportedMediaType:
		return problemDefinition{status: status, code: openapi.UNSUPPORTEDMEDIATYPE, title: "Unsupported media type", detail: "Use application/json for this request."}
	case http.StatusServiceUnavailable:
		return problemDefinition{status: status, code: openapi.UNAVAILABLE, title: "Service unavailable", detail: "A required dependency is unavailable."}
	default:
		return problemDefinition{status: http.StatusInternalServerError, code: openapi.INTERNAL, title: "Internal server error", detail: "The server could not complete the request."}
	}
}
