package testutil

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func Logger(ctx context.Context, t *testing.T) context.Context {
	logger := zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = testWriter{t}
		w.TimeFormat = time.RFC3339
	})).With().Timestamp().Caller().Logger()
	return logger.WithContext(ctx)
}

type testWriter struct {
	t *testing.T
}

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.t.Logf("%s", bytes.TrimSpace(p))
	return len(p), nil
}
