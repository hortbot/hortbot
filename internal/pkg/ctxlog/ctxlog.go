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
// logger is returned.
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

// FromContextWith gets a logger from a context, adds the specified fields,
// and returns context with the new logger.
func FromContextWith(ctx context.Context, fields ...zap.Field) (context.Context, *zap.Logger) {
	if len(fields) == 0 {
		return ctx, FromContext(ctx)
	}

	logger := FromContext(ctx).With(fields...)
	return WithLogger(ctx, logger), logger
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
