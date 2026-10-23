package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mmmIlia/badgoodtune/internal/domain"
	"github.com/mmmIlia/badgoodtune/internal/usecase"
)

type businessObserver struct {
	users, logins, albums int
	scores                []int
}

func (o *businessObserver) UserRegistered()         { o.users++ }
func (o *businessObserver) LoginSucceeded()         { o.logins++ }
func (o *businessObserver) AlbumCreated()           { o.albums++ }
func (o *businessObserver) ReviewCreated(score int) { o.scores = append(o.scores, score) }

type authRepositoryFake struct {
	usecase.AuthRepository
	err error
}

func (f authRepositoryFake) CreateUserWithSession(context.Context, string, string, usecase.NewSession) (domain.User, error) {
	return domain.User{ID: "user"}, f.err
}
func (f authRepositoryFake) GetCredentialByUsername(context.Context, string) (usecase.Credential, error) {
	return usecase.Credential{User: domain.User{ID: "user"}, PasswordHash: "hash"}, nil
}
func (f authRepositoryFake) CreateSession(context.Context, domain.ID, usecase.NewSession) error {
	return f.err
}

type passwordFake struct{ matches bool }

func (passwordFake) Hash(context.Context, string) (string, error)            { return "hash", nil }
func (f passwordFake) Matches(context.Context, string, string) (bool, error) { return f.matches, nil }
func (passwordFake) DummyHash() string                                       { return "dummy" }

type tokenFake struct{}

func (tokenFake) New() (string, []byte, error) { return "token", []byte("digest"), nil }
func (tokenFake) Digest(string) []byte         { return []byte("digest") }

func TestAuthObservesOnlyCommittedSuccess(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		matches bool
	}{
		{name: "success", matches: true},
		{name: "storage error", err: errors.New("commit failed"), matches: true},
		{name: "invalid password"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := &businessObserver{}
			s, err := usecase.NewAuthService(authRepositoryFake{err: tc.err}, passwordFake{matches: tc.matches}, tokenFake{}, fixedClock{now: time.Now()}, time.Hour, o)
			require.NoError(t, err)
			_, err = s.Register(context.Background(), "listener", "a-long-password")
			if tc.err != nil {
				require.Error(t, err)
				assert.Zero(t, o.users)
			} else {
				require.NoError(t, err)
				assert.Equal(t, 1, o.users)
			}
			assert.Zero(t, o.logins, "registration is not an explicit login")
			_, err = s.Login(context.Background(), "listener", "a-long-password")
			if tc.err != nil || !tc.matches {
				require.Error(t, err)
				assert.Zero(t, o.logins)
			} else {
				require.NoError(t, err)
				assert.Equal(t, 1, o.logins)
			}
		})
	}
}

func TestAlbumObservesOnlySuccessfulCreate(t *testing.T) {
	for _, tc := range []struct {
		name  string
		actor domain.ID
		draft domain.AlbumDraft
		err   error
		want  int
	}{
		{name: "success", actor: "user", draft: domain.AlbumDraft{Artist: "Portishead", Title: "Dummy", ReleaseYear: 1994, Genre: "Trip hop"}, want: 1},
		{name: "unauthenticated"},
		{name: "invalid draft", actor: "user"},
		{name: "conflict", actor: "user", draft: domain.AlbumDraft{Artist: "Portishead", Title: "Dummy", ReleaseYear: 1994, Genre: "Trip hop"}, err: domain.ErrConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := &businessObserver{}
			s, err := usecase.NewAlbumService(&albumRepositoryFake{createError: tc.err}, fixedClock{now: time.Now()}, o)
			require.NoError(t, err)
			_, err = s.Create(context.Background(), tc.actor, tc.draft)
			if tc.want == 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.want, o.albums)
		})
	}
}

func TestReviewObservesPublicationButNotEditsOrDeletion(t *testing.T) {
	o := &businessObserver{}
	repository := &reviewRepositoryFake{review: domain.Review{ID: "review", AlbumID: "album", Author: domain.User{ID: "author"}, Headline: "Headline", Body: "Review", Score: 9, Version: 1}}
	s, err := usecase.NewReviewService(repository, &albumRepositoryFake{album: domain.Album{ID: "album"}}, fixedClock{now: time.Now()}, o)
	require.NoError(t, err)
	draft := domain.ReviewDraft{Headline: "Headline", Body: "Review", Score: 9}
	_, err = s.Create(context.Background(), "author", "album", draft)
	require.NoError(t, err)
	assert.Equal(t, []int{9}, o.scores)

	repository.createError = domain.ErrConflict
	_, err = s.Create(context.Background(), "author", "album", draft)
	require.ErrorIs(t, err, domain.ErrConflict)
	draft.Score = 11
	_, err = s.Create(context.Background(), "author", "album", draft)
	require.Error(t, err)
	score := 8
	_, err = s.Update(context.Background(), "author", "album", "review", domain.ReviewPatch{Score: &score, Version: 1})
	require.NoError(t, err)
	require.NoError(t, s.Delete(context.Background(), "author", "album", "review", 2))
	assert.Equal(t, []int{9}, o.scores)
}
