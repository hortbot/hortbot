package bnsq

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type atomicDuration struct {
	v atomic.Int64
}

func (a *atomicDuration) Load() time.Duration {
	return time.Duration(a.v.Load())
}

func (a *atomicDuration) Store(d time.Duration) {
	a.v.Store(int64(d))
}

var testingSleep atomicDuration

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
