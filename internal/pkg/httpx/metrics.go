package httpx

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	labelClientName = "client_name"
	labelCode       = "code"
	labelMethod     = "method"
)

var (
	labelNames = []string{
		labelClientName,
		labelCode,
		labelMethod,
	}

	metricRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "httpx",
		Name:      "request_total",
		Help:      "Total number of requests messages.",
	}, labelNames)

	metricErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "hortbot",
		Subsystem: "httpx",
		Name:      "request_error_total",
		Help:      "Total number of request errors.",
	}, labelNames)

	metricRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "hortbot",
		Subsystem: "httpx",
		Name:      "request_duration_seconds",
		Help:      "Duration of request handling.",
		Buckets:   []float64{.00025, .0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, labelNames)
)

// These functions are copied from promhttp, with modifications.

func makeLabels(clientName string, reqMethod string, status int) prometheus.Labels {
	return prometheus.Labels{
		labelClientName: clientName,
		labelMethod:     sanitizeMethod(reqMethod),
		labelCode:       sanitizeCode(status),
	}
}

func sanitizeMethod(m string) string {
	switch m {
	case "GET", "get":
		return "get"
	case "PUT", "put":
		return "put"
	case "HEAD", "head":
		return "head"
	case "POST", "post":
		return "post"
	case "DELETE", "delete":
		return "delete"
	case "CONNECT", "connect":
		return "connect"
	case "OPTIONS", "options":
		return "options"
	case "NOTIFY", "notify":
		return "notify"
	default:
		return strings.ToLower(m)
	}
}

//nolint:gocyclo
func sanitizeCode(s int) string {
	switch s {
	case 0:
		return "0" // error

	case 100:
		return "100"
	case 101:
		return "101"

	case 200:
		return "200"
	case 201:
		return "201"
	case 202:
		return "202"
	case 203:
		return "203"
	case 204:
		return "204"
	case 205:
		return "205"
	case 206:
		return "206"

	case 300:
		return "300"
	case 301:
		return "301"
	case 302:
		return "302"
	case 304:
		return "304"
	case 305:
		return "305"
	case 307:
		return "307"

	case 400:
		return "400"
	case 401:
		return "401"
	case 402:
		return "402"
	case 403:
		return "403"
	case 404:
		return "404"
	case 405:
		return "405"
	case 406:
		return "406"
	case 407:
		return "407"
	case 408:
		return "408"
	case 409:
		return "409"
	case 410:
		return "410"
	case 411:
		return "411"
	case 412:
		return "412"
	case 413:
		return "413"
	case 414:
		return "414"
	case 415:
		return "415"
	case 416:
		return "416"
	case 417:
		return "417"
	case 418:
		return "418"

	case 500:
		return "500"
	case 501:
		return "501"
	case 502:
		return "502"
	case 503:
		return "503"
	case 504:
		return "504"
	case 505:
		return "505"

	case 428:
		return "428"
	case 429:
		return "429"
	case 431:
		return "431"
	case 511:
		return "511"

	default:
		return strconv.Itoa(s)
	}
}
