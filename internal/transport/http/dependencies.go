package httpapi

import (
	"context"
	"log/slog"
	"time"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

type AuthUseCases interface {
	Register(ctx context.Context, username, password string) (usecase.Session, error)
	Login(ctx context.Context, username, password string) (usecase.Session, error)
	Authenticate(ctx context.Context, plainToken string) (domain.User, error)
	Logout(ctx context.Context, plainToken string) error
}

type AlbumUseCases interface {
	Create(ctx context.Context, actorID domain.ID, input domain.AlbumDraft) (domain.Album, error)
	Get(ctx context.Context, albumID domain.ID) (domain.Album, error)
	List(ctx context.Context, request usecase.AlbumListRequest) (usecase.AlbumPage, error)
	Update(ctx context.Context, actorID, albumID domain.ID, patch domain.AlbumPatch) (domain.Album, error)
	Delete(ctx context.Context, actorID, albumID domain.ID, expectedVersion int64) error
}

type ReviewUseCases interface {
	Create(ctx context.Context, actorID, albumID domain.ID, input domain.ReviewDraft) (domain.Review, error)
	Get(ctx context.Context, albumID, reviewID domain.ID) (domain.Review, error)
	List(ctx context.Context, request usecase.ReviewListRequest) (usecase.ReviewPage, error)
	Update(ctx context.Context, actorID, albumID, reviewID domain.ID, patch domain.ReviewPatch) (domain.Review, error)
	Delete(ctx context.Context, actorID, albumID, reviewID domain.ID, expectedVersion int64) error
}

type ReadinessProbe interface {
	Ping(ctx context.Context) error
}

type Dependencies struct {
	Auth     AuthUseCases
	Albums   AlbumUseCases
	Reviews  ReviewUseCases
	Probe    ReadinessProbe
	Logger   *slog.Logger
	Observer HTTPObserver
}

type HTTPObserver interface {
	RequestStarted()
	RequestCompleted(operation string, status int, duration time.Duration)
}

type Options struct {
	CookieName       string
	CookieSecure     bool
	ReadinessTimeout time.Duration
}
