package bot

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var metricMessagesDropped = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "hortbot",
	Subsystem: "bot_service",
	Name:      "messages_dropped_total",
	Help:      "Total number of queued messages intentionally removed without handling.",
}, []string{"reason"})
