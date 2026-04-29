package logger

import (
	"context"
	"errors"
	"log/slog"
)

// isContextCancellation reports whether err is a context cancellation or
// deadline-exceeded error. Such errors usually indicate the caller (HTTP
// client, background job that was cancelled, etc.) went away, not a real
// backend failure, so logging them at ERROR creates alert noise.
func isContextCancellation(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// hasContextCancellationValue scans args for a non-nil error value that is a
// context cancellation. Both `Errorw("msg", "key", err)` style and `Error("msg",
// slog.Any("err", err))` style are supported by walking every arg and testing
// any error-typed value.
func hasContextCancellationValue(args []any) bool {
	for _, a := range args {
		switch v := a.(type) {
		case error:
			if isContextCancellation(v) {
				return true
			}
		case slog.Attr:
			if e, ok := v.Value.Any().(error); ok && isContextCancellation(e) {
				return true
			}
		}
	}
	return false
}

type Interface interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Fatal(msg string, args ...any)
	With(args ...any) Interface
	Named(name string) Interface

	Debugw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})
}

type slogLogger struct {
	logger *slog.Logger
}

func NewLogger() Interface {
	return &slogLogger{
		logger: Get(),
	}
}

func NewLoggerWithSlog(slogLog *slog.Logger) Interface {
	return &slogLogger{
		logger: slogLog,
	}
}

func (l *slogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *slogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...any) {
	// Demote context-cancellation errors to WARN: they indicate caller
	// lifecycle (request canceled, deadline reached) rather than backend
	// failure, so they should not page operators.
	if hasContextCancellationValue(args) {
		l.logger.Warn(msg, args...)
		return
	}
	l.logger.Error(msg, args...)
}

func (l *slogLogger) Fatal(msg string, args ...any) {
	l.logger.Error(msg, args...)
	panic("fatal error")
}

func (l *slogLogger) With(args ...any) Interface {
	return &slogLogger{
		logger: l.logger.With(args...),
	}
}

func (l *slogLogger) Named(name string) Interface {
	return &slogLogger{
		logger: l.logger.With("logger", name),
	}
}

func (l *slogLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.logger.Debug(msg, keysAndValues...)
}

func (l *slogLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.logger.Info(msg, keysAndValues...)
}

func (l *slogLogger) Warnw(msg string, keysAndValues ...interface{}) {
	l.logger.Warn(msg, keysAndValues...)
}

func (l *slogLogger) Errorw(msg string, keysAndValues ...interface{}) {
	// See Error: context-cancellation errors are not real failures.
	if hasContextCancellationValue(keysAndValues) {
		l.logger.Warn(msg, keysAndValues...)
		return
	}
	l.logger.Error(msg, keysAndValues...)
}

func (l *slogLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.logger.Error(msg, keysAndValues...)
	panic("fatal error")
}
