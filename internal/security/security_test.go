package security_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mmmIlia/badgoodtune/internal/security"
)

func TestArgon2HashAndMatch(t *testing.T) {
	t.Parallel()

	hasher, err := security.NewArgon2Hasher(security.Argon2Params{
		MemoryKiB: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32, MaxConcurrent: 1,
	})
	require.NoError(t, err)

	encoded, err := hasher.Hash(context.Background(), "correct horse battery")
	require.NoError(t, err)
	assert.NotContains(t, encoded, "correct horse battery")

	matches, err := hasher.Matches(context.Background(), encoded, "correct horse battery")
	require.NoError(t, err)
	assert.True(t, matches)

	matches, err = hasher.Matches(context.Background(), encoded, "wrong password")
	require.NoError(t, err)
	assert.False(t, matches)
}

func TestArgon2RejectsUnsafeStoredParameters(t *testing.T) {
	t.Parallel()

	hasher, err := security.NewArgon2Hasher(security.Argon2Params{
		MemoryKiB: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32, MaxConcurrent: 1,
	})
	require.NoError(t, err)

	_, err = hasher.Matches(context.Background(), "$argon2id$v=19$m=2000000,t=1,p=1$c2FsdHNhbHQ$a2V5a2V5a2V5a2V5a2V5aw", "password")
	require.Error(t, err)
}

func TestSessionTokensAreRandomAndDigestible(t *testing.T) {
	t.Parallel()

	source := security.SessionTokenSource{}
	first, firstDigest, err := source.New()
	require.NoError(t, err)
	second, secondDigest, err := source.New()
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	assert.Len(t, firstDigest, 32)
	assert.NotEqual(t, firstDigest, secondDigest)
	assert.Equal(t, firstDigest, source.Digest(first))
}
