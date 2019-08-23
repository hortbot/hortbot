package bnsq

import (
	"context"
	"strings"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type nsqLogger struct {
	l *zap.Logger
}

func nsqLoggerFrom(ctx context.Context) nsqLogger {
	return nsqLogger{l: ctxlog.FromContext(ctx)}
}

func (l nsqLogger) Output(calldepth int, s string) error {
	level := zapcore.DebugLevel
	if strings.HasPrefix(s, "ERR") {
		level = zapcore.ErrorLevel
	}

	logger := l.l.WithOptions(zap.AddCallerSkip(calldepth))
	logger.Check(level, s).Write()
	return nil
}
