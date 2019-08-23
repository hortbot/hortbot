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

func newConfig() *nsq.Config {
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

type nsqProducer struct {
	addr     string
	producer *nsq.Producer
	ready    chan struct{}
}

func newProducer(addr string) *nsqProducer {
	return &nsqProducer{
		addr:  addr,
		ready: make(chan struct{}),
	}
}

func (p *nsqProducer) run(ctx context.Context) error {
	producer, err := nsq.NewProducer(p.addr, newConfig())
	if err != nil {
		return err
	}
	defer producer.Stop()

	producer.SetLogger(nsqLoggerFrom(ctx), nsq.LogLevelInfo)

	p.producer = producer
	close(p.ready)

	if err := producer.Ping(); err != nil {
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (p *nsqProducer) get() *nsq.Producer {
	<-p.ready
	return p.producer
}
