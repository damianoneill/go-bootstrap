// pkg/adapter/metrics/prometheus.go
package metrics

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/damianoneill/go-bootstrap/pkg/domain/metrics"
)

type prometheusCollector struct {
	requestDuration *prometheus.HistogramVec
	requestsTotal   *prometheus.CounterVec
	errorsTotal     *prometheus.CounterVec
	reg             prometheus.Registerer
	mu              sync.RWMutex
}

func NewMetricsFactory() metrics.Factory {
	return &PrometheusFactory{}
}

type PrometheusFactory struct{}

func (f *PrometheusFactory) NewCollector(opts ...metrics.Option) (metrics.Collector, error) {
	options := metrics.DefaultOptions()
	for _, opt := range opts {
		if err := opt.ApplyOption(&options); err != nil {
			return nil, fmt.Errorf("applying option: %w", err)
		}
	}

	// Validate options
	if options.ServiceName == "" {
		return nil, fmt.Errorf("service name is required")
	}

	labels := prometheus.Labels{
		"service": options.ServiceName,
	}
	for k, v := range options.Labels {
		labels[k] = v
	}

	buckets := options.Buckets
	if len(buckets) == 0 {
		buckets = prometheus.DefBuckets
	}

	// Validate bucket order
	for i := 1; i < len(buckets); i++ {
		if buckets[i] <= buckets[i-1] {
			return nil, fmt.Errorf("buckets must be in increasing order: %v", buckets)
		}
	}

	c := &prometheusCollector{
		reg: prometheus.DefaultRegisterer,
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "http_request_duration_seconds",
				Help:        "HTTP request duration in seconds",
				Buckets:     buckets,
				ConstLabels: labels,
			},
			[]string{"method", "path", "status"},
		),
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "http_requests_total",
				Help:        "Total number of HTTP requests",
				ConstLabels: labels,
			},
			[]string{"method", "path", "status"},
		),
		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "http_errors_total",
				Help:        "Total number of HTTP errors",
				ConstLabels: labels,
			},
			[]string{"method", "path", "status"},
		),
	}

	// Register all collectors
	collectors := []prometheus.Collector{
		c.requestDuration,
		c.requestsTotal,
		c.errorsTotal,
	}

	for _, collector := range collectors {
		if err := c.reg.Register(collector); err != nil {
			// Clean up any already registered collectors
			for _, col := range collectors {
				c.reg.Unregister(col)
			}
			return nil, fmt.Errorf("registering collector: %w", err)
		}
	}

	return c, nil
}

func (c *prometheusCollector) CollectRequestMetrics(method, path string, status int, duration float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	labels := prometheus.Labels{
		"method": method,
		"path":   path,
		"status": fmt.Sprintf("%d", status),
	}

	c.requestDuration.With(labels).Observe(duration)
	c.requestsTotal.With(labels).Inc()

	if status >= 400 {
		c.errorsTotal.With(labels).Inc()
	}
}

func (c *prometheusCollector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.reg.Unregister(c.requestDuration)
	c.reg.Unregister(c.requestsTotal)
	c.reg.Unregister(c.errorsTotal)

	return nil
}
