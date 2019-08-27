package tracing

import (
	"io"

	jaegerzap "github.com/jaegertracing/jaeger-client-go/log/zap"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Init initializes the global tracer.
func Init(name string, debug bool, logger *zap.Logger) (io.Closer, error) {
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		return nil, err
	}

	if debug {
		cfg.Sampler = &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		}
		cfg.Reporter = &jaegercfg.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: cfg.Reporter.LocalAgentHostPort,
		}
	}

	logger = logger.WithOptions(wrapCoreWithLevel(zap.WarnLevel))

	jLogger := jaegerzap.NewLogger(logger)
	jMetricsFactory := metrics.NullFactory

	return cfg.InitGlobalTracer(name, jaegercfg.Logger(jLogger), jaegercfg.Metrics(jMetricsFactory))
}

// github.com/uber-go/zap/issues/581
func wrapCoreWithLevel(level zapcore.Level) zap.Option {
	return zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return &coreWithLevel{
			Core:  core,
			level: level,
		}
	})
}

type coreWithLevel struct {
	zapcore.Core
	level zapcore.Level
}

func (c *coreWithLevel) Enabled(level zapcore.Level) bool {
	return c.level.Enabled(level) && c.Core.Enabled(level)
}

func (c *coreWithLevel) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.level.Enabled(e.Level) {
		return ce
	}
	return c.Core.Check(e, ce)
}
