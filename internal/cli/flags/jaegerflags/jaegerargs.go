// Package jaegerflags processes Jaeger-related flags.
package jaegerflags

import (
	"context"

	"contrib.go.opencensus.io/exporter/jaeger"
	"contrib.go.opencensus.io/integrations/ocsql"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

type Jaeger struct {
	Agent string `long:"jaeger-agent" env:"HB_JAEGER_AGENT" description:"jaeger agent address"`
}

var DefaultJaeger = Jaeger{}

func (args *Jaeger) Init(ctx context.Context, name string, debug bool) func() {
	if args.Agent == "" {
		return func() {}
	}

	exporter, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: args.Agent,
		Process: jaeger.Process{
			ServiceName: name,
		},
	})
	if err != nil {
		ctxlog.Fatal(ctx, "error creating jaeger exporter", zap.Error(err))
	}
	trace.RegisterExporter(exporter)

	if debug {
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	}

	return exporter.Flush
}

func (args *Jaeger) DriverName(ctx context.Context, driverName string, debug bool) string {
	if args.Agent == "" {
		return driverName
	}

	driverName, err := ocsql.Register(driverName, ocsql.WithAllTraceOptions(), ocsql.WithQueryParams(debug))
	if err != nil {
		ctxlog.Fatal(ctx, "error registering osql driver", zap.Error(err))
	}
	return driverName
}
