package rstf

import (
	"log/slog"
	"os"
)

// Logger provides structured, request-scoped logging.
type Logger struct {
	slog *slog.Logger
}

// NewLogger creates a Logger that writes JSON to stdout.
func NewLogger() *Logger {
	return &Logger{
		slog: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

// Info logs at INFO level.
func (l *Logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

// Warn logs at WARN level.
func (l *Logger) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
}

// Error logs at ERROR level.
func (l *Logger) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
}

// Debug logs at DEBUG level.
func (l *Logger) Debug(msg string, args ...any) {
	l.slog.Debug(msg, args...)
}

// With returns a new Logger with the given key-value pairs attached to every log entry.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{slog: l.slog.With(args...)}
}

