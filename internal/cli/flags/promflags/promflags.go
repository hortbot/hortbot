// Package promflags provides prometheus metric flags.
package promflags

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

// Prometheus contains Prometheus flags.
type Prometheus struct {
	Enabled bool `long:"prometheus-enabled" env:"HB_PROMETHEUS_ENABLED" description:"Enable Prometheus exporting"`
}

// Default contains the default flags. Make a copy of this, do not reuse.
var Default = Prometheus{}

// Run runs the Prometheus trace endpoint in a background goroutine which exits
// when the context is canceled.
func (args *Prometheus) Run(ctx context.Context) {
	if !args.Enabled {
		return
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	go func() {
		srv := http.Server{
			Addr:              ":2112",
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		}
		if err := srv.ListenAndServe(); err != nil {
			ctxlog.Error(ctx, "prometheus server error", zap.Error(err))
		}
	}()
}
