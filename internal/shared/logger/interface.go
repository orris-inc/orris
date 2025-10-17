package logger

import "go.uber.org/zap"

// Interface represents a logger interface for dependency injection
type Interface interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	With(fields ...zap.Field) Interface
	Named(name string) Interface
	
	// Sugar logger methods for easier usage
	Debugw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})
}

// zapLogger implements Interface
type zapLogger struct {
	logger *zap.Logger
}

// NewLogger creates a new logger instance
func NewLogger() Interface {
	return &zapLogger{
		logger: Get(),
	}
}

// NewLoggerWithZap creates a new logger instance with existing zap logger
func NewLoggerWithZap(zapLog *zap.Logger) Interface {
	return &zapLogger{
		logger: zapLog,
	}
}

// Debug implements Interface
func (l *zapLogger) Debug(msg string, fields ...zap.Field) {
	l.logger.Debug(msg, fields...)
}

// Info implements Interface
func (l *zapLogger) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

// Warn implements Interface
func (l *zapLogger) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

// Error implements Interface
func (l *zapLogger) Error(msg string, fields ...zap.Field) {
	l.logger.Error(msg, fields...)
}

// Fatal implements Interface
func (l *zapLogger) Fatal(msg string, fields ...zap.Field) {
	l.logger.Fatal(msg, fields...)
}

// With implements Interface
func (l *zapLogger) With(fields ...zap.Field) Interface {
	return &zapLogger{
		logger: l.logger.With(fields...),
	}
}

// Named implements Interface
func (l *zapLogger) Named(name string) Interface {
	return &zapLogger{
		logger: l.logger.Named(name),
	}
}

// Debugw implements Interface (sugar logger)
func (l *zapLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.logger.Sugar().Debugw(msg, keysAndValues...)
}

// Infow implements Interface (sugar logger)
func (l *zapLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.logger.Sugar().Infow(msg, keysAndValues...)
}

// Warnw implements Interface (sugar logger)
func (l *zapLogger) Warnw(msg string, keysAndValues ...interface{}) {
	l.logger.Sugar().Warnw(msg, keysAndValues...)
}

// Errorw implements Interface (sugar logger)
func (l *zapLogger) Errorw(msg string, keysAndValues ...interface{}) {
	l.logger.Sugar().Errorw(msg, keysAndValues...)
}

// Fatalw implements Interface (sugar logger)
func (l *zapLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.logger.Sugar().Fatalw(msg, keysAndValues...)
}