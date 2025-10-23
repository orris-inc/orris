package logger

import "log/slog"

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
	l.logger.Error(msg, keysAndValues...)
}

func (l *slogLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.logger.Error(msg, keysAndValues...)
	panic("fatal error")
}
