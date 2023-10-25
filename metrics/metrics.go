package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

const traceIDKey = "trace_id" // https://opentelemetry.io/docs/specs/otel/metrics/data-model/#exemplars

type Metrics struct {
	searchesTotal   prometheus.Counter
	searchesFailed  prometheus.Counter
	searchesLatency prometheus.Histogram

	insertsTotal   prometheus.Counter
	insertsFailed  prometheus.Counter
	insertsLatency prometheus.Histogram

	deletesTotal   prometheus.Counter
	deletesFailed  prometheus.Counter
	deletesLatency prometheus.Histogram

	server *http.Server
}

// New creates a new Prometheus Metrics instance, with its HTTP server registered on the input port.
func New(port int) (*Metrics, error) {
	if port < 0 {
		port = 0
	}

	promMetrics := newProm()

	reg, err := promMetrics.Registry()
	if err != nil {
		return nil, err
	}

	promMetrics.server = newServer(port, reg)

	return promMetrics, nil
}
