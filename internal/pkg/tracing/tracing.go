package tracing

import (
	"contrib.go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"
)

// Init initializes the global tracer.
func Init(name string, agent string, debug bool) (func(), error) {
	exporter, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: agent,
		Process: jaeger.Process{
			ServiceName: name,
		},
	})
	if err != nil {
		return nil, err
	}
	trace.RegisterExporter(exporter)

	if debug {
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	}

	return exporter.Flush, nil
}
