package logger

import (
	"context"
	"log/slog"
	"runtime"
)

type conditionalSourceHandler struct {
	handler          slog.Handler
	showSourceLevels map[slog.Level]bool
}

// NewConditionalSourceHandler wraps a handler to conditionally show source location
// based on log level. Source location is only shown for the specified levels.
// This is useful for reducing log volume in production while maintaining debuggability
// for important log levels.
//
// The wrapped handler should have AddSource: false in its options.
// This wrapper will conditionally add source attributes for specified levels.
//
// Example:
//
//	handler := NewConditionalSourceHandler(
//	    tint.NewHandler(os.Stdout, opts),
//	    slog.LevelWarn,
//	    slog.LevelError,
//	)
func NewConditionalSourceHandler(handler slog.Handler, showSourceForLevels ...slog.Level) slog.Handler {
	levelMap := make(map[slog.Level]bool)
	for _, level := range showSourceForLevels {
		levelMap[level] = true
	}
	return &conditionalSourceHandler{
		handler:          handler,
		showSourceLevels: levelMap,
	}
}

func (h *conditionalSourceHandler) Handle(ctx context.Context, r slog.Record) error {
	// If this level should show source, add it manually
	if h.showSourceLevels[r.Level] {
		// Get the source location - skip this frame and one more for the slog internal frame
		var pcs [1]uintptr
		runtime.Callers(3, pcs[:])
		fs := runtime.CallersFrames(pcs[:])
		f, _ := fs.Next()

		// Create a Source value for this location
		source := &slog.Source{
			Function: f.Function,
			File:     f.File,
			Line:     f.Line,
		}

		// Add source attribute to the record
		r.AddAttrs(slog.Attr{
			Key:   slog.SourceKey,
			Value: slog.AnyValue(source),
		})
	}

	return h.handler.Handle(ctx, r)
}

func (h *conditionalSourceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &conditionalSourceHandler{
		handler:          h.handler.WithAttrs(attrs),
		showSourceLevels: h.showSourceLevels,
	}
}

func (h *conditionalSourceHandler) WithGroup(name string) slog.Handler {
	return &conditionalSourceHandler{
		handler:          h.handler.WithGroup(name),
		showSourceLevels: h.showSourceLevels,
	}
}

func (h *conditionalSourceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}
