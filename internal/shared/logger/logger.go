package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"golang.org/x/term"

	"orris/internal/infrastructure/config"
)

var (
	Logger      *slog.Logger
	atomicLevel *slog.LevelVar
)

func Init(cfg *config.LoggerConfig) error {
	atomicLevel = new(slog.LevelVar)
	level := slog.LevelInfo
	if cfg.Level != "" {
		switch strings.ToLower(cfg.Level) {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}
	atomicLevel.Set(level)

	var writer io.Writer
	switch strings.ToLower(cfg.OutputPath) {
	case "stdout", "":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		file, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		writer = file
	}

	// Determine which log levels should show source location
	// By default: warn and error show source, debug and info don't (production-friendly)
	showSourceLevels := []slog.Level{slog.LevelWarn, slog.LevelError}
	serverCfg := config.Get().Server
	if serverCfg.Mode == "debug" {
		// In debug mode, show source for all levels
		showSourceLevels = []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}
	}

	var handler slog.Handler

	if cfg.Format == "json" {
		baseHandler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level:     atomicLevel,
			AddSource: false,
		})
		handler = NewConditionalSourceHandler(baseHandler, showSourceLevels...)
	} else {
		noColor := !isTerminal(writer)

		tintOpts := &tint.Options{
			Level:      atomicLevel,
			TimeFormat: time.DateTime,
			AddSource:  false,
			NoColor:    noColor,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == "error" && a.Value.Kind() == slog.KindAny {
					if err, ok := a.Value.Any().(error); ok {
						return tint.Err(err)
					}
				}
				return a
			},
		}
		baseHandler := tint.NewHandler(writer, tintOpts)
		handler = NewConditionalSourceHandler(baseHandler, showSourceLevels...)
	}

	Logger = slog.New(handler)
	slog.SetDefault(Logger)

	return nil
}

func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func SetLevel(level slog.Level) {
	if atomicLevel != nil {
		atomicLevel.Set(level)
	}
}

func Get() *slog.Logger {
	if Logger == nil {
		noColor := !term.IsTerminal(int(os.Stdout.Fd()))

		baseHandler := tint.NewHandler(os.Stdout, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.DateTime,
			AddSource:  true,
			NoColor:    noColor,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == "error" && a.Value.Kind() == slog.KindAny {
					if err, ok := a.Value.Any().(error); ok {
						return tint.Err(err)
					}
				}
				return a
			},
		})
		// Default: show source for warn and error only (production-friendly)
		handler := NewConditionalSourceHandler(baseHandler, slog.LevelWarn, slog.LevelError)
		Logger = slog.New(handler)
		slog.SetDefault(Logger)
	}
	return Logger
}

func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	Get().Error(msg, args...)
	os.Exit(1)
}

func Sync() error {
	return nil
}

func WithFields(args ...any) *slog.Logger {
	return Get().With(args...)
}

func WithComponent(component string) *slog.Logger {
	return Get().With("component", component)
}

func Named(name string) *slog.Logger {
	return Get().With("logger", name)
}
