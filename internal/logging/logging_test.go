package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerWritesNormalizedJSON(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := New(&output, "info", "badgoodtune-test")
	logger.Info("request completed", slog.String("request_id", "request-1"))

	var event map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &event))
	assert.NotEmpty(t, event["timestamp"])
	assert.Equal(t, "info", event["level"])
	assert.Equal(t, "request completed", event["message"])
	assert.Equal(t, "badgoodtune-test", event["service.name"])
	assert.Equal(t, "request-1", event["request_id"])
	assert.NotContains(t, event, "time")
	assert.NotContains(t, event, "msg")
}

func TestLoggerHonorsConfiguredLevel(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := New(&output, "warn", "badgoodtune-test")
	logger.Info("not emitted")
	logger.Warn("emitted")

	var event map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &event))
	assert.Equal(t, "warn", event["level"])
	assert.Equal(t, "emitted", event["message"])
}
