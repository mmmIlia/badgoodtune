package logging

import (
	"io"
	"log/slog"
	"strings"
)

func New(output io.Writer, level, serviceName string) *slog.Logger {
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level:       slogLevel,
		ReplaceAttr: normalizeAttribute,
	})
	return slog.New(handler).With(slog.String("service.name", serviceName))
}

func normalizeAttribute(_ []string, attribute slog.Attr) slog.Attr {
	switch attribute.Key {
	case slog.TimeKey:
		attribute.Key = "timestamp"
	case slog.LevelKey:
		attribute.Value = slog.StringValue(strings.ToLower(attribute.Value.String()))
	case slog.MessageKey:
		attribute.Key = "message"
	}
	return attribute
}
