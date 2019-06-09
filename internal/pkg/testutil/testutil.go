package testutil

import (
	"bytes"
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Tester interface {
	Helper()
	Logf(format string, args ...interface{})
}

func Logger(t Tester) *zap.Logger {
	return buildLogger(Writer{T: t})
}

type Writer struct {
	T Tester
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
