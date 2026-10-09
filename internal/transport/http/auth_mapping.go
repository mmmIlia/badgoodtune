package httpapi

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/transport/http/openapi"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

func userView(user domain.User) (openapi.User, error) {
	id, err := uuid.Parse(user.ID.String())
	if err != nil {
		return openapi.User{}, fmt.Errorf("map user ID: %w", err)
	}
	return openapi.User{
		Id:        id,
		Username:  user.Username,
		CreatedAt: user.CreatedAt.UTC(),
	}, nil
}

func sessionView(session usecase.Session) (openapi.SessionView, error) {
	user, err := userView(session.User)
	if err != nil {
		return openapi.SessionView{}, err
	}
	return openapi.SessionView{User: user, ExpiresAt: session.ExpiresAt.UTC()}, nil
}
