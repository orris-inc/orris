package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"orris/internal/infrastructure/config"
)

// CallerDisplayMode represents how caller information is displayed
type CallerDisplayMode int

const (
	// CallerShort shows only filename:line (e.g., "server.go:67")
	CallerShort CallerDisplayMode = iota
	// CallerFull shows full path (e.g., "orris/cmd/orris/commands/server.go:67")
	CallerFull
	// CallerNone disables caller display
	CallerNone
)

const (
	// callerPadding defines the padding for caller information alignment
	callerPadding = 30
)

var (
	Logger            *zap.Logger
	Sugar             *zap.SugaredLogger
	atomicLevel       zap.AtomicLevel
	callerDisplayMode = CallerShort // Default to most concise
)

// Init initializes the global logger based on configuration
func Init(cfg *config.LoggerConfig) error {
	// Parse log level
	atomicLevel = zap.NewAtomicLevel()
	level := zapcore.InfoLevel
	if cfg.Level != "" {
		if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
			return err
		}
	}
	atomicLevel.SetLevel(level)

	// Configure encoder
	var encoderConfig zapcore.EncoderConfig
	if cfg.Format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		// Use custom caller encoder for alignment
		if callerDisplayMode != CallerNone {
			encoderConfig.EncodeCaller = customCallerEncoder
		} else {
			encoderConfig.CallerKey = ""
		}
	}
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Configure output
	var writeSyncer zapcore.WriteSyncer
	switch strings.ToLower(cfg.OutputPath) {
	case "stdout", "":
		writeSyncer = zapcore.AddSync(os.Stdout)
	case "stderr":
		writeSyncer = zapcore.AddSync(os.Stderr)
	default:
		file, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		writeSyncer = zapcore.AddSync(file)
	}

	// Create encoder
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create core and logger
	core := zapcore.NewCore(encoder, writeSyncer, atomicLevel)
	
	// Build logger with options
	opts := []zap.Option{
		zap.AddStacktrace(zapcore.ErrorLevel),
	}
	
	// Add caller if not disabled
	if callerDisplayMode != CallerNone {
		opts = append(opts, zap.AddCaller(), zap.AddCallerSkip(1))
	}
	
	Logger = zap.New(core, opts...)
	Sugar = Logger.Sugar()

	return nil
}

// SetCallerDisplayMode changes how caller information is displayed
func SetCallerDisplayMode(mode CallerDisplayMode) {
	callerDisplayMode = mode
	// Re-initialize if logger exists
	if Logger != nil {
		// Note: This requires re-initialization to take effect
		// You may want to store the config and re-init here
	}
}

// SetLevel changes the log level dynamically
func SetLevel(level zapcore.Level) {
	if atomicLevel.Level() != level {
		atomicLevel.SetLevel(level)
	}
}

// Get returns the global logger instance
func Get() *zap.Logger {
	if Logger == nil {
		// Fallback to development logger if not initialized
		Logger, _ = zap.NewDevelopment(zap.AddCallerSkip(1))
		Sugar = Logger.Sugar()
	}
	return Logger
}

// GetSugar returns the sugared logger instance
func GetSugar() *zap.SugaredLogger {
	if Sugar == nil {
		Get() // This will initialize both Logger and Sugar
	}
	return Sugar
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// Sync flushes any buffered log entries
func Sync() error {
	if Logger != nil {
		return Logger.Sync()
	}
	return nil
}

// WithFields returns a logger with additional fields
func WithFields(fields ...zap.Field) *zap.Logger {
	return Get().With(fields...)
}

// WithComponent returns a logger with component field
func WithComponent(component string) *zap.Logger {
	return Get().With(zap.String("component", component))
}

// Named adds a sub-scope to the logger's name
func Named(name string) *zap.Logger {
	return Get().Named(name)
}

// formatCallerPath formats caller path based on display mode with alignment
func formatCallerPath(caller zapcore.EntryCaller) string {
	if !caller.Defined {
		return "undefined"
	}

	var callerStr string
	switch callerDisplayMode {
	case CallerShort:
		// Show only filename:line
		callerStr = fmt.Sprintf("%s:%d", filepath.Base(caller.File), caller.Line)
	case CallerFull:
		// Show full path:line
		callerStr = fmt.Sprintf("%s:%d", caller.File, caller.Line)
	default:
		return ""
	}

	// Apply padding for alignment
	if len(callerStr) < callerPadding {
		return callerStr + strings.Repeat(" ", callerPadding-len(callerStr))
	}
	return callerStr
}

// customCallerEncoder creates a custom caller encoder with alignment
func customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(formatCallerPath(caller))
}