package testutil

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/hortbot/hortbot/internal/ctxlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Logger(ctx context.Context, t *testing.T) context.Context {
	logger := buildLogger(testWriter{t})
	return ctxlog.WithLogger(ctx, logger)
}

type testWriter struct {
	t *testing.T
}

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.t.Helper()
	tw.t.Logf("%s", bytes.TrimSpace(p))
	return len(p), nil
}

func buildLogger(w io.Writer) *zap.Logger {
	encConf := zap.NewDevelopmentEncoderConfig()
	encConf.EncodeLevel = zapcore.CapitalColorLevelEncoder
	enc := zapcore.NewConsoleEncoder(encConf)
	ws := zapcore.Lock(zapcore.AddSync(w))
	level := zap.NewAtomicLevelAt(zap.DebugLevel)

	opts := []zap.Option{
		zap.ErrorOutput(ws),
		zap.Development(),
		zap.AddCaller(),
		zap.AddStacktrace(zap.WarnLevel),
	}

	return zap.New(
		zapcore.NewCore(enc, ws, level),
		opts...,
	)
}
