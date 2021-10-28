package bot

import (
	"context"

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

	metricDuplicateMessage = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "duplicate_total",
		Help:      "Total number of duplicate messages ignored.",
	})

	metricHandleError = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "handle_error_total",
		Help:      "Total number of handler errors.",
	})

	metricHandleTimingFromTwitch = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "handle_timing_from_twitch_seconds",
		Help:      "Time from Twitch to the message queue.",
		Buckets:   []float64{.00025, .0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})

	metricHandleTimingInQueue = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "handle_timing_in_queue_seconds",
		Help:      "Time in NSQ to handler.",
		Buckets:   []float64{.00025, .0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})

	metricHandleTimingBegin = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "handle_timing_begin_seconds",
		Help:      "Time spent in beginning transaction.",
		Buckets:   []float64{.00025, .0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})

	metricHandleTimingHandle = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "handle_timing_handle_seconds",
		Help:      "Time spent in actual handler function.",
		Buckets:   []float64{.00025, .0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})

	metricHandleTimingCommit = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "handle_timing_commit_seconds",
		Help:      "Time spent committing transaction.",
		Buckets:   []float64{.00025, .0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})

	metricRepeatedError = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "repeated_error_total",
		Help:      "Total number of repeated command errors.",
	})

	metricScheduledError = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "bot",
		Name:      "scheduled_error_total",
		Help:      "Total number of scheduled command errors.",
	})
)

func setMetricRepeatGauges(ctx context.Context, rep *repeat.Repeater) {
	r, s, err := rep.Count(ctx)
	if err != nil {
		return
	}
	metricRepeatGauge.Set(float64(r))
	metricScheduleGauge.Set(float64(s))
}
