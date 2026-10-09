package httpapi

import (
	"context"

	"github.com/mmmIlia/badgoodtune/internal/transport/http/openapi"
)

func (s *Server) GetLiveness(context.Context, openapi.GetLivenessRequestObject) (openapi.GetLivenessResponseObject, error) {
	return openapi.GetLiveness200JSONResponse{Status: openapi.Ok}, nil
}

func (s *Server) GetReadiness(ctx context.Context, _ openapi.GetReadinessRequestObject) (openapi.GetReadinessResponseObject, error) {
	probeContext, cancel := context.WithTimeout(ctx, s.readinessTimeout)
	defer cancel()
	if err := s.probe.Ping(probeContext); err != nil {
		return nil, unavailableError{cause: err}
	}
	return openapi.GetReadiness200JSONResponse{Status: openapi.Ok}, nil
}
