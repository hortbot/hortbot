package bot

import (
	"github.com/hortbot/hortbot/internal/pkg/repeat"
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

	metricHandleDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "handle_duration_seconds",
		Help:      "Duration of message handling.",
		Buckets:   []float64{.00025, .0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"irc_command"})

	metricRepeatGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "active_repeated",
		Help:      "Total number of active repeated commands.",
	})

	metricScheduleGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "active_scheduled",
		Help:      "Total number of active scheduled commands.",
	})
)

func setMetricRepeatGauges(rep *repeat.Repeater) {
	r, s := rep.Count()
	metricRepeatGauge.Set(float64(r))
	metricScheduleGauge.Set(float64(s))
}
