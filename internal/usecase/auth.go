package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

type Session struct {
	User       domain.User
	ExpiresAt  time.Time
	PlainToken string
}

type AuthService struct {
	repository AuthRepository
	passwords  PasswordHasher
	tokens     SessionTokenSource
	clock      Clock
	sessionTTL time.Duration
	observer   AuthObserver
}

func NewAuthService(
	repository AuthRepository,
	passwords PasswordHasher,
	tokens SessionTokenSource,
	clock Clock,
	sessionTTL time.Duration,
	observer AuthObserver,
) (*AuthService, error) {
	if observer == nil {
		return nil, errors.New("auth observer is required")
	}
	if repository == nil {
		return nil, errors.New("auth repository is required")
	}
	if passwords == nil {
		return nil, errors.New("password hasher is required")
	}
	if tokens == nil {
		return nil, errors.New("session token source is required")
	}
	if clock == nil {
		return nil, errors.New("clock is required")
	}
	if sessionTTL <= 0 {
		return nil, errors.New("session TTL must be positive")
	}

	return &AuthService{
		repository: repository,
		passwords:  passwords,
		tokens:     tokens,
		clock:      clock,
		sessionTTL: sessionTTL,
		observer:   observer,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, username, password string) (Session, error) {
	normalizedUsername, err := domain.ValidateRegistration(username, password)
	if err != nil {
		return Session{}, err
	}

	passwordHash, err := s.passwords.Hash(ctx, password)
	if err != nil {
		return Session{}, fmt.Errorf("hash password: %w", err)
	}
	plainToken, session, err := s.newSession()
	if err != nil {
		return Session{}, err
	}
	user, err := s.repository.CreateUserWithSession(ctx, normalizedUsername, passwordHash, session)
	if err != nil {
		return Session{}, fmt.Errorf("create user and session: %w", err)
	}

	s.observer.UserRegistered()
	return Session{User: user, ExpiresAt: session.ExpiresAt, PlainToken: plainToken}, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (Session, error) {
	normalizedUsername, err := domain.ValidateLogin(username, password)
	if err != nil {
		return Session{}, err
	}

	credential, err := s.repository.GetCredentialByUsername(ctx, normalizedUsername)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return Session{}, fmt.Errorf("get credential: %w", err)
		}
		if _, matchErr := s.passwords.Matches(ctx, s.passwords.DummyHash(), password); matchErr != nil {
			return Session{}, fmt.Errorf("verify dummy password: %w", matchErr)
		}
		return Session{}, domain.ErrInvalidCredentials
	}

	matches, err := s.passwords.Matches(ctx, credential.PasswordHash, password)
	if err != nil {
		return Session{}, fmt.Errorf("verify password: %w", err)
	}
	if !matches {
		return Session{}, domain.ErrInvalidCredentials
	}

	plainToken, session, err := s.newSession()
	if err != nil {
		return Session{}, err
	}
	if err := s.repository.CreateSession(ctx, credential.User.ID, session); err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}

	s.observer.LoginSucceeded()
	return Session{User: credential.User, ExpiresAt: session.ExpiresAt, PlainToken: plainToken}, nil
}

func (s *AuthService) newSession() (string, NewSession, error) {
	plainToken, tokenHash, err := s.tokens.New()
	if err != nil {
		return "", NewSession{}, fmt.Errorf("generate session token: %w", err)
	}
	now := s.clock.Now().UTC()
	return plainToken, NewSession{
		TokenHash: tokenHash,
		ExpiresAt: now.Add(s.sessionTTL),
		CreatedAt: now,
	}, nil
}

func (s *AuthService) Authenticate(ctx context.Context, plainToken string) (domain.User, error) {
	if plainToken == "" {
		return domain.User{}, domain.ErrUnauthenticated
	}

	user, err := s.repository.GetUserBySession(ctx, s.tokens.Digest(plainToken), s.clock.Now().UTC())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, domain.ErrUnauthenticated
		}
		return domain.User{}, fmt.Errorf("get user by session: %w", err)
	}
	return user, nil
}

func (s *AuthService) Logout(ctx context.Context, plainToken string) error {
	if plainToken == "" {
		return nil
	}
	if err := s.repository.DeleteSession(ctx, s.tokens.Digest(plainToken)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
