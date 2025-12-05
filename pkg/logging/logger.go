// Package logging provides a structured logging interface.
package logging

import (
	"context"
	"log/slog"
)

// Logger provides structured logging capabilities.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

// SlogLogger wraps slog.Logger to implement our Logger interface.
type SlogLogger struct {
	logger *slog.Logger
}

// New creates a new logger wrapping a slog.Logger.
func New(logger *slog.Logger) (l Logger) {
	l = &SlogLogger{logger: logger}
	return l
}

// Info logs an info level message.
func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn logs a warning level message.
func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error logs an error level message.
func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// InfoContext logs an info level message with context.
func (l *SlogLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, args...)
}

// WarnContext logs a warning level message with context.
func (l *SlogLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, args...)
}

// ErrorContext logs an error level message with context.
func (l *SlogLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, args...)
}
