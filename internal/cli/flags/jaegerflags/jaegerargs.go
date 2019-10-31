// Package jaegerflags processes Jaeger-related flags.
package jaegerflags

import (
	"context"
	"database/sql/driver"

	"contrib.go.opencensus.io/exporter/jaeger"
	"contrib.go.opencensus.io/integrations/ocsql"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

type Jaeger struct {
	JaegerAgent string `long:"jaeger-agent" env:"HB_JAEGER_AGENT" description:"jaeger agent address"`
}

var DefaultJaeger = Jaeger{}

func (args *Jaeger) InitJaeger(ctx context.Context, name string, debug bool) func() {
	if args.JaegerAgent == "" {
		return func() {}
	}

	exporter, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: args.JaegerAgent,
		Process: jaeger.Process{
			ServiceName: name,
		},
	})
	if err != nil {
		ctxlog.FromContext(ctx).Fatal("error creating jaeger exporter", zap.Error(err))
	}
	trace.RegisterExporter(exporter)

	if debug {
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	}

	return exporter.Flush
}

func (args *Jaeger) TraceDB(debug bool, d driver.Connector) driver.Connector {
	if args.JaegerAgent == "" {
		return d
	}
	return ocsql.WrapConnector(d, ocsql.WithAllTraceOptions(), ocsql.WithQueryParams(debug))
}
