package logger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

// newCapturingLogger builds a slogLogger that writes JSON to buf at LevelDebug
// so we can assert what level a given call landed on.
func newCapturingLogger(buf *bytes.Buffer) *slogLogger {
	h := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return &slogLogger{logger: slog.New(h)}
}

func levelOf(t *testing.T, line string) string {
	t.Helper()
	// slog JSON output includes "level":"INFO|WARN|ERROR|DEBUG"
	for _, want := range []string{`"level":"DEBUG"`, `"level":"INFO"`, `"level":"WARN"`, `"level":"ERROR"`} {
		if strings.Contains(line, want) {
			return strings.Trim(strings.SplitN(want, ":", 2)[1], `"`)
		}
	}
	t.Fatalf("no level found in log line: %s", line)
	return ""
}

func TestErrorw_DemotesContextCanceled(t *testing.T) {
	var buf bytes.Buffer
	log := newCapturingLogger(&buf)

	log.Errorw("query failed", "id", 42, "error", context.Canceled)

	if got := levelOf(t, buf.String()); got != "WARN" {
		t.Fatalf("expected WARN for context.Canceled, got %s; line=%s", got, buf.String())
	}
}

func TestErrorw_DemotesDeadlineExceeded(t *testing.T) {
	var buf bytes.Buffer
	log := newCapturingLogger(&buf)

	log.Errorw("query failed", "error", context.DeadlineExceeded)

	if got := levelOf(t, buf.String()); got != "WARN" {
		t.Fatalf("expected WARN for context.DeadlineExceeded, got %s", got)
	}
}

func TestErrorw_DemotesWrappedCancellation(t *testing.T) {
	var buf bytes.Buffer
	log := newCapturingLogger(&buf)

	wrapped := fmt.Errorf("db query failed: %w", context.Canceled)
	log.Errorw("repo failed", "error", wrapped)

	if got := levelOf(t, buf.String()); got != "WARN" {
		t.Fatalf("expected WARN for wrapped cancellation, got %s", got)
	}
}

func TestErrorw_PreservesRealError(t *testing.T) {
	var buf bytes.Buffer
	log := newCapturingLogger(&buf)

	log.Errorw("real failure", "error", errors.New("connection refused"))

	if got := levelOf(t, buf.String()); got != "ERROR" {
		t.Fatalf("expected ERROR for real error, got %s", got)
	}
}

func TestError_DemotesContextCancellationViaSlogAttr(t *testing.T) {
	var buf bytes.Buffer
	log := newCapturingLogger(&buf)

	log.Error("query failed", slog.Any("err", context.Canceled))

	if got := levelOf(t, buf.String()); got != "WARN" {
		t.Fatalf("expected WARN for slog.Any(context.Canceled), got %s", got)
	}
}

func TestError_PreservesRealErrorWithSlogAttr(t *testing.T) {
	var buf bytes.Buffer
	log := newCapturingLogger(&buf)

	log.Error("query failed", slog.Any("err", errors.New("boom")))

	if got := levelOf(t, buf.String()); got != "ERROR" {
		t.Fatalf("expected ERROR for real slog.Any error, got %s", got)
	}
}

func TestErrorw_NoCancellationKeepsError(t *testing.T) {
	var buf bytes.Buffer
	log := newCapturingLogger(&buf)

	log.Errorw("just a message", "id", 7)

	if got := levelOf(t, buf.String()); got != "ERROR" {
		t.Fatalf("expected ERROR when no error value present, got %s", got)
	}
}
