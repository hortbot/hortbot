package web

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var metricRequest = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "hortbot",
	Subsystem: "web",
	Name:      "request_total",
	Help:      "Total number of HTTP requests.",
}, []string{"code", "method"})
