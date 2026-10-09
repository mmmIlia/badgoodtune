package architecture_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mmmIlia/badgoodtune/internal/domain"
)

func TestAPIContractMatchesDomainLimits(t *testing.T) {
	t.Parallel()

	document, err := openapi3.NewLoader().LoadFromFile(filepath.Join(repositoryRoot(t), "api", "openapi.yaml"))
	require.NoError(t, err)
	require.NoError(t, document.Validate(context.Background()))

	assertStringLimits(t, document, "RegisterUserRequest", "username", domain.UsernameMinLength, domain.UsernameMaxLength)
	assertStringLimits(t, document, "RegisterUserRequest", "password", domain.PasswordMinLength, domain.PasswordMaxLength)
	assertStringLimits(t, document, "CreateAlbumRequest", "artist", 1, domain.AlbumTextMaxLength)
	assertStringLimits(t, document, "CreateAlbumRequest", "title", 1, domain.AlbumTextMaxLength)
	assertStringLimits(t, document, "CreateAlbumRequest", "genre", 1, domain.GenreMaxLength)
	assertStringLimits(t, document, "CreateReviewRequest", "headline", 1, domain.ReviewHeadlineMaxLength)
	assertStringLimits(t, document, "CreateReviewRequest", "body", 1, domain.ReviewBodyMaxLength)

	releaseYear := schemaProperty(t, document, "CreateAlbumRequest", "releaseYear")
	require.NotNil(t, releaseYear.Min)
	assert.Equal(t, float64(domain.FirstRecordedYear), *releaseYear.Min)
	assert.Nil(t, releaseYear.Max)
	assert.NotEmpty(t, releaseYear.Description)

	score := schemaProperty(t, document, "CreateReviewRequest", "score")
	require.NotNil(t, score.Min)
	require.NotNil(t, score.Max)
	assert.Equal(t, float64(domain.ReviewScoreMin), *score.Min)
	assert.Equal(t, float64(domain.ReviewScoreMax), *score.Max)
}

func assertStringLimits(
	t *testing.T,
	document *openapi3.T,
	schemaName, propertyName string,
	minLength, maxLength int,
) {
	t.Helper()
	property := schemaProperty(t, document, schemaName, propertyName)
	assert.Equal(t, uint64(minLength), property.MinLength)
	require.NotNil(t, property.MaxLength)
	assert.Equal(t, uint64(maxLength), *property.MaxLength)
}

func schemaProperty(t *testing.T, document *openapi3.T, schemaName, propertyName string) *openapi3.Schema {
	t.Helper()
	schema, ok := document.Components.Schemas[schemaName]
	require.True(t, ok, "schema %s is missing", schemaName)
	require.NotNil(t, schema.Value)
	property, ok := schema.Value.Properties[propertyName]
	require.True(t, ok, "property %s.%s is missing", schemaName, propertyName)
	require.NotNil(t, property.Value)
	return property.Value
}
