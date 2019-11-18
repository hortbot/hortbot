package bnsq

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricPublished = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "bnsq",
		Name:      "published_total",
		Help:      "Total number of published NSQ messages.",
	}, []string{"topic"})

	metricHandled = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "bnsq",
		Name:      "handled_total",
		Help:      "Total number of handled NSQ messages.",
	}, []string{"topic"})
)
