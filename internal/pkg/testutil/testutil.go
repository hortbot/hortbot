// Package testutil provides useful testing helpers.
package testutil

import (
	"bytes"
	"io"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// HelpLogger is an interface with Helper and Logf functions.
type HelpLogger interface {
	Helper()
	Logf(format string, args ...any)
}

// Logger returns a zap logger which logs to a test, and a stop function.
// The caller must defer the stop function to prevent panics from background
// goroutines that log after the test has exited.
func Logger(t HelpLogger) (*zap.Logger, func()) {
	w := &stoppableWriter{inner: Writer{T: t}}
	return buildLogger(w), w.Stop
}

// Writer is an io.Writer which writes to a test logger.
type Writer struct {
	T HelpLogger
}

func (tw Writer) Write(p []byte) (n int, err error) {
	tw.T.Helper()
	tw.T.Logf("%s", bytes.TrimSpace(p))
	return len(p), nil
}

// stoppableWriter wraps a Writer with an atomic flag so writes are
// silently discarded once Stop is called.
type stoppableWriter struct {
	stopped atomic.Bool
	inner   Writer
}

func (w *stoppableWriter) Write(p []byte) (n int, err error) {
	if w.stopped.Load() {
		return len(p), nil
	}
	return w.inner.Write(p)
}

func (w *stoppableWriter) Stop() {
	w.stopped.Store(true)
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
