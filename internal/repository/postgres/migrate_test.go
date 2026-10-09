package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mmmIlia/badgoodtune/internal/repository/postgres"
)

func TestMigrateRequiresDatabase(t *testing.T) {
	t.Parallel()

	err := postgres.Migrate(context.Background(), nil)
	assert.ErrorContains(t, err, "database is required")
}
