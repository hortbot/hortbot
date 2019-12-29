// Package ctxlog provides context-scoped logging.
package ctxlog

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a new zap Logger.
func New(debug bool) *zap.Logger {
	var logConfig zap.Config

	if debug {
		logConfig = zap.NewDevelopmentConfig()
		logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		logConfig = zap.NewProductionConfig()
	}

	logger, err := logConfig.Build()
	if err != nil {
		panic(err)
	}

	return logger
}

type loggerKey struct{}

var nopLogger = zap.NewNop()

// FromContext gets a zap logger from a context. If none is set, then a nop
// logger is returned. The Debug, Error, With, WithOptions, etc, functions
// should be preferred over using this function to access a logger.
func FromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
		return logger
	}
	return nopLogger
}

// WithLogger adds a zap logger to a context.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// Debug writes a log item with the debug level.
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

// Error writes a log item with the error level.
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

// Fatal writes a log item with the fatal level.
func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).WithOptions(zap.AddCallerSkip(1)).Fatal(msg, fields...)
}

// Info writes a log item with the info level.
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

// Warn writes a log item with the warning level.
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

// With returns a new context with the proivded fields.
func With(ctx context.Context, fields ...zap.Field) context.Context {
	return WithLogger(ctx, FromContext(ctx).With(fields...))
}

// WithOptions returns a context containing a logger with the specified options.
func WithOptions(ctx context.Context, opts ...zap.Option) context.Context {
	return WithLogger(ctx, FromContext(ctx).WithOptions(opts...))
}

type noTrace struct{}

func (noTrace) Enabled(zapcore.Level) bool { return false }

// NoTrace disables stack traces in log events.
func NoTrace() zap.Option {
	return zap.AddStacktrace(noTrace{})
}

type plainError struct {
	e error
}

func (pe plainError) Error() string {
	return pe.e.Error()
}

// PlainError is like zap.Error, but won't also include extra debugging info
// (which is duplicated in some logging levels).
func PlainError(err error) zap.Field {
	return zap.Error(plainError{e: err})
}
