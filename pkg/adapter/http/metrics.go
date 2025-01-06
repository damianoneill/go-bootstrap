package http

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

// metrics holds Prometheus metrics collectors for HTTP monitoring
type metrics struct {
	requestDuration *prometheus.HistogramVec
	requestsTotal   *prometheus.CounterVec
	errorsTotal     *prometheus.CounterVec
}

func newMetrics(serviceName string) *metrics {
	commonLabels := prometheus.Labels{
		"service": serviceName,
	}

	m := &metrics{
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "http_request_duration_seconds",
				Help:        "HTTP request duration in seconds",
				Buckets:     prometheus.DefBuckets,
				ConstLabels: commonLabels,
			},
			[]string{"method", "path", "status"},
		),
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "http_requests_total",
				Help:        "Total number of HTTP requests",
				ConstLabels: commonLabels,
			},
			[]string{"method", "path", "status"},
		),
		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "http_errors_total",
				Help:        "Total number of HTTP errors",
				ConstLabels: commonLabels,
			},
			[]string{"method", "path", "status"},
		),
	}

	// Register metrics with Prometheus
	prometheus.MustRegister(
		m.requestDuration,
		m.requestsTotal,
		m.errorsTotal,
	)

	return m
}

// collectMetrics records metrics if the path is not excluded
func (m *metrics) collectMetrics(method, path string, status int, duration float64) {
	labels := prometheus.Labels{
		"method": method,
		"path":   path,
		"status": fmt.Sprintf("%d", status),
	}

	m.requestDuration.With(labels).Observe(duration)
	m.requestsTotal.With(labels).Inc()

	if status >= http.StatusBadRequest {
		m.errorsTotal.With(labels).Inc()
	}
}
