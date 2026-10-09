package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/mmmIlia/badgoodtune/internal/transport/http/openapi"
)

func (s *Server) contractHandlers() (api http.Handler, health http.Handler, err error) {
	spec, err := openapi.GetSwagger()
	if err != nil {
		return nil, nil, fmt.Errorf("load embedded OpenAPI contract: %w", err)
	}
	spec.Servers = nil
	cookieAuth, ok := spec.Components.SecuritySchemes["cookieAuth"]
	if !ok || cookieAuth.Value == nil {
		return nil, nil, errors.New("OpenAPI contract has no cookieAuth security scheme")
	}
	cookieAuth.Value.Name = s.cookieName

	strict := openapi.NewStrictHandlerWithOptions(
		s,
		[]openapi.StrictMiddlewareFunc{s.recordOperation},
		openapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				s.writeProblem(w, r, requestDecodeError{cause: err})
			},
			ResponseErrorHandlerFunc: s.writeProblem,
		},
	)
	generated := openapi.HandlerWithOptions(strict, openapi.StdHTTPServerOptions{
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			s.writeProblem(w, r, requestDecodeError{cause: err})
		},
	})

	validator := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		Options: openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
		ErrorHandlerWithOpts: func(
			ctx context.Context,
			err error,
			w http.ResponseWriter,
			r *http.Request,
			opts nethttpmiddleware.ErrorHandlerOpts,
		) {
			s.writeContractProblem(w, r, err, opts.StatusCode)
		},
		DoNotValidateServers: true,
	})

	contract := validator(generated)
	api = s.limitRequestBody(s.requireJSONContentType(s.loadSession(contract)))
	return api, contract, nil
}
