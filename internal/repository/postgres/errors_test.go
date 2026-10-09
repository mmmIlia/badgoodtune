package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

func TestMapWriteErrorMapsKnownConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		code       string
		constraint string
		want       error
	}{
		{name: "album identity", code: uniqueViolation, constraint: albumsIdentityConstraint, want: domain.ErrConflict},
		{name: "review album", code: foreignKeyViolation, constraint: reviewsAlbumConstraint, want: domain.ErrNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := mapWriteError(&pgconn.PgError{Code: test.code, ConstraintName: test.constraint})
			assert.ErrorIs(t, err, test.want)
		})
	}
}

func TestMapWriteErrorPreservesUnknownConstraints(t *testing.T) {
	t.Parallel()

	databaseError := &pgconn.PgError{Code: uniqueViolation, ConstraintName: "unexpected_constraint"}
	assert.True(t, errors.Is(mapWriteError(databaseError), databaseError))
}
