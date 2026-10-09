package usecase

import (
	"context"
	"time"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

type PasswordHasher interface {
	Hash(ctx context.Context, password string) (string, error)
	Matches(ctx context.Context, encodedHash, password string) (bool, error)
	DummyHash() string
}

type SessionTokenSource interface {
	New() (plainToken string, digest []byte, err error)
	Digest(plainToken string) []byte
}

type Credential struct {
	User         domain.User
	PasswordHash string
}

type NewSession struct {
	TokenHash []byte
	ExpiresAt time.Time
	CreatedAt time.Time
}

type AuthRepository interface {
	CreateUserWithSession(
		ctx context.Context,
		username string,
		passwordHash string,
		session NewSession,
	) (domain.User, error)
	GetCredentialByUsername(ctx context.Context, username string) (Credential, error)
	CreateSession(ctx context.Context, userID domain.ID, session NewSession) error
	GetUserBySession(ctx context.Context, tokenHash []byte, now time.Time) (domain.User, error)
	DeleteSession(ctx context.Context, tokenHash []byte) error
}
