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

	metricCommands = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "commands_total",
		Help:      "Total number of handled commands.",
	})

	metricAutoreplies = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "autoreply_total",
		Help:      "Total number of executed autoreplies.",
	})

	metricRepeated = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "repeated_total",
		Help:      "Total number of executed repeated commands.",
	})

	metricScheduled = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "scheduled_total",
		Help:      "Total number of executed scheduled commands.",
	})
)
