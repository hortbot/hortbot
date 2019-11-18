package bot

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricHandled = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "handled_total",
		Help:      "Total number of handled messages.",
	})
)
