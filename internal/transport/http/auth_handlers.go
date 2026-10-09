package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/mmmIlia/badgoodtune/internal/transport/http/openapi"
)

func (s *Server) RegisterUser(ctx context.Context, request openapi.RegisterUserRequestObject) (openapi.RegisterUserResponseObject, error) {
	if request.Body == nil || request.Body.Password == nil {
		return nil, missingBodyField("password")
	}
	session, err := s.auth.Register(ctx, request.Body.Username, *request.Body.Password)
	if err != nil {
		return nil, err
	}
	view, err := sessionView(session)
	if err != nil {
		return nil, err
	}
	return openapi.RegisterUser201JSONResponse{
		Body: view,
		Headers: openapi.RegisterUser201ResponseHeaders{
			Location:  stringPointer("/api/v1/users/me"),
			SetCookie: stringPointer(s.sessionCookie(session.PlainToken, session.ExpiresAt).String()),
		},
	}, nil
}

func (s *Server) CreateSession(ctx context.Context, request openapi.CreateSessionRequestObject) (openapi.CreateSessionResponseObject, error) {
	if request.Body == nil || request.Body.Password == nil {
		return nil, missingBodyField("password")
	}
	session, err := s.auth.Login(ctx, request.Body.Username, *request.Body.Password)
	if err != nil {
		return nil, err
	}
	view, err := sessionView(session)
	if err != nil {
		return nil, err
	}
	return openapi.CreateSession201JSONResponse{
		Body: view,
		Headers: openapi.CreateSession201ResponseHeaders{
			Location:  stringPointer("/api/v1/sessions/current"),
			SetCookie: stringPointer(s.sessionCookie(session.PlainToken, session.ExpiresAt).String()),
		},
	}, nil
}

func (s *Server) DeleteCurrentSession(ctx context.Context, _ openapi.DeleteCurrentSessionRequestObject) (openapi.DeleteCurrentSessionResponseObject, error) {
	state := sessionStateFromContext(ctx)
	if err := s.auth.Logout(ctx, state.token); err != nil {
		return nil, err
	}
	return openapi.DeleteCurrentSession204Response{Headers: openapi.DeleteCurrentSession204ResponseHeaders{
		SetCookie: stringPointer(s.expiredSessionCookie().String()),
	}}, nil
}

func (s *Server) GetCurrentUser(ctx context.Context, _ openapi.GetCurrentUserRequestObject) (openapi.GetCurrentUserResponseObject, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	view, err := userView(user)
	if err != nil {
		return nil, err
	}
	return openapi.GetCurrentUser200JSONResponse(view), nil
}

func (s *Server) sessionCookie(token string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     s.cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt.UTC(),
		MaxAge:   max(1, int(time.Until(expiresAt).Seconds())),
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	}
}

func (s *Server) expiredSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     s.cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	}
}
