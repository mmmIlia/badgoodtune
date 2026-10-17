package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/transport/http/openapi"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

const (
	testUserID  = domain.ID("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	testAlbumID = domain.ID("6ba7b811-9dad-11d1-80b4-00c04fd430c8")
)

func TestRegisterUserSetsSecureSessionProperties(t *testing.T) {
	handler := newTestHandler(t, testDependencies{})

	response := performJSONRequest(t, handler, http.MethodPost, "/api/v1/users", `{
		"username":"listener_1",
		"password":"a-long-password"
	}`, nil)

	require.Equal(t, http.StatusCreated, response.Code)
	assert.Equal(t, "/api/v1/users/me", response.Header().Get("Location"))
	cookies := response.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "test_session", cookies[0].Name)
	assert.Equal(t, "plain-session-token", cookies[0].Value)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, cookies[0].SameSite)
	assert.Equal(t, "/", cookies[0].Path)

	var body map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.NotContains(t, body, "token")
}

func TestProtectedOperationRequiresSession(t *testing.T) {
	handler := newTestHandler(t, testDependencies{})

	response := performJSONRequest(t, handler, http.MethodPost, "/api/v1/albums", `{
		"artist":"Massive Attack",
		"title":"Mezzanine",
		"releaseYear":1998,
		"genre":"Trip-hop"
	}`, nil)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	problem := decodeProblem(t, response)
	assert.Equal(t, openapi.UNAUTHENTICATED, problem.Code)
	assert.NotEmpty(t, problem.RequestId)
}

func TestDomainValidationIsReturnedAsProblemDetails(t *testing.T) {
	albums := &albumUseCasesStub{createError: domain.NewValidationError(
		domain.Violation{Field: "releaseYear", Description: "is outside the supported range"},
	)}
	handler := newTestHandler(t, testDependencies{albums: albums})

	response := performJSONRequest(t, handler, http.MethodPost, "/api/v1/albums", `{
		"artist":"Future Artist",
		"title":"Future Album",
		"releaseYear":2027,
		"genre":"Ambient"
	}`, &http.Cookie{Name: "test_session", Value: "valid"})

	require.Equal(t, http.StatusBadRequest, response.Code)
	problem := decodeProblem(t, response)
	require.NotNil(t, problem.Violations)
	assert.Equal(t, []openapi.FieldViolation{{Field: "releaseYear", Description: "is outside the supported range"}}, *problem.Violations)
}

func TestOpenAPIRejectsUnknownFieldsAndWrongMediaType(t *testing.T) {
	handler := newTestHandler(t, testDependencies{})

	unknownField := performJSONRequest(t, handler, http.MethodPost, "/api/v1/users", `{
		"username":"listener_1",
		"password":"a-long-password",
		"admin":true
	}`, nil)
	require.Equal(t, http.StatusBadRequest, unknownField.Code)
	assert.Equal(t, openapi.INVALIDARGUMENT, decodeProblem(t, unknownField).Code)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString("not-json"))
	request.Header.Set("Content-Type", "text/plain")
	wrongMediaType := httptest.NewRecorder()
	handler.ServeHTTP(wrongMediaType, request)
	require.Equal(t, http.StatusUnsupportedMediaType, wrongMediaType.Code)
	assert.Equal(t, openapi.UNSUPPORTEDMEDIATYPE, decodeProblem(t, wrongMediaType).Code)
}

func TestCrossOriginMutationIsRejected(t *testing.T) {
	handler := newTestHandler(t, testDependencies{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(`{
		"username":"listener_1",
		"password":"a-long-password"
	}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://attacker.example")
	request.Host = "badgoodtune.example"
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code)
	assert.Equal(t, openapi.PERMISSIONDENIED, decodeProblem(t, response).Code)
}

func TestHealthAndUnknownRouteResponses(t *testing.T) {
	handler := newTestHandler(t, testDependencies{})

	live := httptest.NewRecorder()
	handler.ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	require.Equal(t, http.StatusOK, live.Code)
	assert.NotEmpty(t, live.Header().Get("X-Request-ID"))
	assert.Equal(t, "nosniff", live.Header().Get("X-Content-Type-Options"))

	unknown := httptest.NewRecorder()
	handler.ServeHTTP(unknown, httptest.NewRequest(http.MethodGet, "/albums/example", nil))
	assert.Equal(t, http.StatusNotFound, unknown.Code)
}

func TestHealthDoesNotLoadUserSession(t *testing.T) {
	handler := newTestHandler(t, testDependencies{auth: healthAuthStub{}})
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	request.AddCookie(&http.Cookie{Name: "test_session", Value: "valid"})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
}

func TestReadinessReportsDependencyFailure(t *testing.T) {
	handler := newTestHandler(t, testDependencies{probe: probeStub{err: errors.New("database offline")}})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	assert.Equal(t, openapi.UNAVAILABLE, decodeProblem(t, response).Code)
}

func TestRequestLogLevel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, slog.LevelDebug, requestLogLevel("/health/live", http.StatusOK))
	assert.Equal(t, slog.LevelInfo, requestLogLevel("/api/v1/albums", http.StatusOK))
	assert.Equal(t, slog.LevelWarn, requestLogLevel("/api/v1/albums", http.StatusUnauthorized))
	assert.Equal(t, slog.LevelError, requestLogLevel("/api/v1/albums", http.StatusInternalServerError))
}

type testDependencies struct {
	auth    AuthUseCases
	albums  AlbumUseCases
	reviews ReviewUseCases
	probe   ReadinessProbe
}

func newTestHandler(t *testing.T, custom testDependencies) http.Handler {
	t.Helper()
	if custom.auth == nil {
		custom.auth = authUseCasesStub{}
	}
	if custom.albums == nil {
		custom.albums = &albumUseCasesStub{}
	}
	if custom.reviews == nil {
		custom.reviews = reviewUseCasesStub{}
	}
	if custom.probe == nil {
		custom.probe = probeStub{}
	}
	handler, err := NewHandler(Dependencies{
		Auth:    custom.auth,
		Albums:  custom.albums,
		Reviews: custom.reviews,
		Probe:   custom.probe,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, Options{
		CookieName:       "test_session",
		ReadinessTimeout: time.Second,
	})
	require.NoError(t, err)
	return handler
}

func performJSONRequest(t *testing.T, handler http.Handler, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeProblem(t *testing.T, response *httptest.ResponseRecorder) openapi.Problem {
	t.Helper()
	var problem openapi.Problem
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem), response.Body.String())
	return problem
}

type authUseCasesStub struct{}

type healthAuthStub struct{ authUseCasesStub }

func (healthAuthStub) Authenticate(context.Context, string) (domain.User, error) {
	panic("health endpoint attempted authentication")
}

func (authUseCasesStub) Register(context.Context, string, string) (usecase.Session, error) {
	return usecase.Session{User: testUser(), ExpiresAt: time.Now().Add(time.Hour), PlainToken: "plain-session-token"}, nil
}

func (authUseCasesStub) Login(context.Context, string, string) (usecase.Session, error) {
	return usecase.Session{User: testUser(), ExpiresAt: time.Now().Add(time.Hour), PlainToken: "plain-session-token"}, nil
}

func (authUseCasesStub) Authenticate(_ context.Context, token string) (domain.User, error) {
	if token != "valid" {
		return domain.User{}, domain.ErrUnauthenticated
	}
	return testUser(), nil
}

func (authUseCasesStub) Logout(context.Context, string) error { return nil }

type albumUseCasesStub struct{ createError error }

func (s *albumUseCasesStub) Create(context.Context, domain.ID, domain.AlbumDraft) (domain.Album, error) {
	if s.createError != nil {
		return domain.Album{}, s.createError
	}
	return domain.Album{ID: testAlbumID, Owner: testUser(), Artist: "Artist", Title: "Album", ReleaseYear: 2000, Genre: "Rock", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func (*albumUseCasesStub) Get(context.Context, domain.ID) (domain.Album, error) {
	return domain.Album{}, domain.ErrNotFound
}

func (*albumUseCasesStub) List(context.Context, usecase.AlbumListRequest) (usecase.AlbumPage, error) {
	return usecase.AlbumPage{Albums: []domain.Album{}}, nil
}

func (*albumUseCasesStub) Update(context.Context, domain.ID, domain.ID, domain.AlbumPatch) (domain.Album, error) {
	return domain.Album{}, domain.ErrNotFound
}

func (*albumUseCasesStub) Delete(context.Context, domain.ID, domain.ID, int64) error {
	return domain.ErrNotFound
}

type reviewUseCasesStub struct{}

func (reviewUseCasesStub) Create(context.Context, domain.ID, domain.ID, domain.ReviewDraft) (domain.Review, error) {
	return domain.Review{}, domain.ErrNotFound
}

func (reviewUseCasesStub) Get(context.Context, domain.ID, domain.ID) (domain.Review, error) {
	return domain.Review{}, domain.ErrNotFound
}

func (reviewUseCasesStub) List(context.Context, usecase.ReviewListRequest) (usecase.ReviewPage, error) {
	return usecase.ReviewPage{Reviews: []domain.Review{}}, nil
}

func (reviewUseCasesStub) Update(context.Context, domain.ID, domain.ID, domain.ID, domain.ReviewPatch) (domain.Review, error) {
	return domain.Review{}, domain.ErrNotFound
}

func (reviewUseCasesStub) Delete(context.Context, domain.ID, domain.ID, domain.ID, int64) error {
	return domain.ErrNotFound
}

type probeStub struct{ err error }

func (p probeStub) Ping(context.Context) error { return p.err }

func testUser() domain.User {
	return domain.User{ID: testUserID, Username: "listener_1", CreatedAt: time.Now().UTC()}
}
