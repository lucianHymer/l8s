package logging

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"
)

type contextKey string

const loggerKey contextKey = "logger"

var defaultLogger *slog.Logger

func init() {
	defaultLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, time.Now().Format(time.RFC3339))
			}
			return a
		},
	}))
}

func SetDefault(logger *slog.Logger) {
	defaultLogger = logger
}

func Default() *slog.Logger {
	return defaultLogger
}

func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return defaultLogger
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func WithError(err error) slog.Attr {
	return slog.String("error", err.Error())
}

func WithField(key string, value any) slog.Attr {
	return slog.Any(key, value)
}

func CallerInfo(skip int) (function string, file string, line int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", "unknown", 0
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown", file, line
	}
	return fn.Name(), file, line
}