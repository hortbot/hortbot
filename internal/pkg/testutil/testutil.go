// Package testutil provides useful testing helpers.
package testutil

import (
	"bytes"
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// HelpLogger is an interface with Helper and Logf functions.
type HelpLogger interface {
	Helper()
	Logf(format string, args ...any)
}

// Logger returns a zap logger which logs to a test.
func Logger(t HelpLogger) *zap.Logger {
	return buildLogger(Writer{T: t})
}

// Writer is a io.Writer which writes to a test logger.
type Writer struct {
	T HelpLogger
}

func (tw Writer) Write(p []byte) (n int, err error) {
	tw.T.Helper()
	tw.T.Logf("%s", bytes.TrimSpace(p))
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
