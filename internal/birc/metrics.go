package birc

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "birc",
		Name:      "received_total",
		Help:      "Total number of received messages.",
	}, []string{"nick"})

	metricSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "birc",
		Name:      "sent_total",
		Help:      "Total number of sent messages.",
	}, []string{"nick"})

	metricReconnects = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "birc",
		Name:      "reconnect_total",
		Help:      "Total number of RECONNECT messages.",
	}, []string{"nick"})

	metricConnect = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "birc",
		Name:      "connect_total",
		Help:      "Total number of connects.",
	}, []string{"nick"})

	metricDisconnect = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "birc",
		Name:      "disconnect_total",
		Help:      "Total number of disconnects.",
	}, []string{"nick"})

	metricSubconns = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "hortbot",
		Subsystem: "birc",
		Name:      "subconns",
		Help:      "Number of disconnects.",
	}, []string{"nick"})
)
