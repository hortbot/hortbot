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

func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).WithOptions(zap.AddCallerSkip(1)).Fatal(msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

func With(ctx context.Context, fields ...zap.Field) context.Context {
	return WithLogger(ctx, FromContext(ctx).With(fields...))
}

func WithOptions(ctx context.Context, opts ...zap.Option) context.Context {
	return WithLogger(ctx, FromContext(ctx).WithOptions(opts...))
}

type noTrace struct{}

func (noTrace) Enabled(zapcore.Level) bool { return false }

func NoTrace() zap.Option {
	return zap.AddStacktrace(noTrace{})
}

type plainError struct {
	e error
}

func (pe plainError) Error() string {
	return pe.e.Error()
}

func PlainError(err error) zap.Field {
	return zap.Error(plainError{e: err})
}
