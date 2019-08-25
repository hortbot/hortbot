package bnsq

import (
	"context"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/nsqio/go-nsq"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var testingSleep time.Duration

func defaultConfig() *nsq.Config {
	config := nsq.NewConfig()
	config.LookupdPollInterval = 5 * time.Second
	return config
}

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
