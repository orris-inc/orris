package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestConditionalSourceHandler(t *testing.T) {
	tests := []struct {
		name             string
		level            slog.Level
		showSourceLevels []slog.Level
		shouldHaveSource bool
	}{
		{
			name:             "INFO without source config",
			level:            slog.LevelInfo,
			showSourceLevels: []slog.Level{slog.LevelWarn, slog.LevelError},
			shouldHaveSource: false,
		},
		{
			name:             "WARN with source config",
			level:            slog.LevelWarn,
			showSourceLevels: []slog.Level{slog.LevelWarn, slog.LevelError},
			shouldHaveSource: true,
		},
		{
			name:             "ERROR with source config",
			level:            slog.LevelError,
			showSourceLevels: []slog.Level{slog.LevelWarn, slog.LevelError},
			shouldHaveSource: true,
		},
		{
			name:             "DEBUG without source config",
			level:            slog.LevelDebug,
			showSourceLevels: []slog.Level{slog.LevelWarn, slog.LevelError},
			shouldHaveSource: false,
		},
		{
			name:             "INFO with explicit source config",
			level:            slog.LevelInfo,
			showSourceLevels: []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError},
			shouldHaveSource: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
				AddSource: false,
			})
			handler := NewConditionalSourceHandler(baseHandler, tt.showSourceLevels...)

			logger := slog.New(handler)
			switch tt.level {
			case slog.LevelDebug:
				logger.Debug("test message")
			case slog.LevelInfo:
				logger.Info("test message")
			case slog.LevelWarn:
				logger.Warn("test message")
			case slog.LevelError:
				logger.Error("test message")
			}

			output := buf.String()
			hasSource := strings.Contains(output, "source=")

			if hasSource != tt.shouldHaveSource {
				t.Errorf("expected source=%v, got %v. Output: %s", tt.shouldHaveSource, hasSource, output)
			}
		})
	}
}

func TestConditionalSourceHandlerWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		AddSource: false,
	})
	handler := NewConditionalSourceHandler(baseHandler, slog.LevelError)

	logger := slog.New(handler).With("user_id", "123")
	logger.Info("test message")

	output := buf.String()
	if strings.Contains(output, "source=") {
		t.Errorf("expected no source for INFO level, but found it. Output: %s", output)
	}
	if !strings.Contains(output, "user_id=123") {
		t.Errorf("expected user_id attribute, but not found. Output: %s", output)
	}
}

func TestConditionalSourceHandlerWithGroup(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		AddSource: false,
	})
	handler := NewConditionalSourceHandler(baseHandler, slog.LevelError)

	logger := slog.New(handler).WithGroup("request")
	logger.Info("test message", "path", "/api/users")

	output := buf.String()
	if strings.Contains(output, "source=") {
		t.Errorf("expected no source for INFO level, but found it. Output: %s", output)
	}
	if !strings.Contains(output, "path") {
		t.Errorf("expected request group with path, but not found. Output: %s", output)
	}
}

func TestConditionalSourceHandlerEnabled(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})
	handler := NewConditionalSourceHandler(baseHandler, slog.LevelError)

	// Should respect the base handler's level
	if !handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected INFO level to be enabled")
	}
	if !handler.Enabled(context.Background(), slog.LevelError) {
		t.Error("expected ERROR level to be enabled")
	}
	if handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected DEBUG level to be disabled")
	}
}
