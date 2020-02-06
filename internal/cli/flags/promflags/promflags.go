// Package promflags provides prometheus metric flags.
package promflags

import (
	"context"
	"net/http"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Prometheus struct {
	Enabled bool `long:"prometheus-enabled" env:"HB_PROMETHEUS_ENABLED" description:"Enable Prometheus exporting"`
}

var Default = Prometheus{}

func (args *Prometheus) Run(ctx context.Context) {
	if !args.Enabled {
		return
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	go func() {
		if err := http.ListenAndServe(":2112", mux); err != nil {
			ctxlog.Error(ctx, "prometheus server error", zap.Error(err))
		}
	}()
}
