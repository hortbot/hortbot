package birc

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricReceived = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "birc",
		Name:      "received_total",
		Help:      "Total number of received messages.",
	})

	metricSent = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "birc",
		Name:      "sent_total",
		Help:      "Total number of sent messages.",
	})

	metricReconnects = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "birc",
		Name:      "reconnect_total",
		Help:      "Total number of RECONNECT messages.",
	})
)
