package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

type sessionContextKey struct{}

type sessionState struct {
	user  *domain.User
	token string
	err   error
}

func (s *Server) loadSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := sessionState{}
		cookie, err := r.Cookie(s.cookieName)
		switch {
		case errors.Is(err, http.ErrNoCookie):
		case err != nil:
			state.err = err
		case cookie.Value == "":
		default:
			state.token = cookie.Value
			user, authErr := s.auth.Authenticate(r.Context(), cookie.Value)
			if authErr == nil {
				state.user = &user
			} else if !errors.Is(authErr, domain.ErrUnauthenticated) {
				state.err = authErr
			}
		}
		ctx := context.WithValue(r.Context(), sessionContextKey{}, state)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func sessionStateFromContext(ctx context.Context) sessionState {
	state, _ := ctx.Value(sessionContextKey{}).(sessionState)
	return state
}

func requireUser(ctx context.Context) (domain.User, error) {
	state := sessionStateFromContext(ctx)
	if state.err != nil {
		return domain.User{}, state.err
	}
	if state.user == nil {
		return domain.User{}, domain.ErrUnauthenticated
	}
	return *state.user, nil
}
