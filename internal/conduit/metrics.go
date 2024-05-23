package conduit

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricHandled = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "conduit",
		Name:      "handled_total",
		Help:      "Total number of handled messages.",
	}, []string{"type"})

	metricDecodeErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "conduit",
		Name:      "decode_error_total",
		Help:      "Total number of erroring message decodes.",
	}, []string{"field", "value"})

metricReconnects = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "conduit",
		Name:      "reconnects_total",
		Help:      "Total number of reconnects.",
	})

	metricDisconnects = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "conduit",
		Name:      "disconnects_total",
		Help:      "Total number of disconnects.",
	})

	metricCreatedSubscriptions = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "conduit",
		Name:      "created_subscriptions_total",
		Help:      "Total number of created subscriptions.",
	})

	metricDeletedSubscriptions = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "conduit",
		Name:      "deleted_subscriptions_total",
		Help:      "Total number of deleted subscriptions.",
	})

	metricSubscriptions = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "hortbot",
		Subsystem: "conduit",
		Name:      "subscriptions",
		Help:      "Total number of subscriptions.",
	})

	metricSubscriptionTypes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "hortbot",
		Subsystem: "conduit",
		Name:      "subscription_types",
		Help:      "Number of subscriptions by type.",
	}, []string{"type"})

	metricWebsockets = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "hortbot",
		Subsystem: "conduit",
		Name:      "websockets",
		Help:      "Number of websockets.",
	})

	metricSyncDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "hortbot",
		Subsystem: "conduit",
		Name:      "sync_duration_seconds",
		Help:      "Duration of sync handling.",
		Buckets:   []float64{.00025, .0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})
)
