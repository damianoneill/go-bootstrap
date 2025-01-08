// pkg/domain/metrics/metrics.go
package metrics

import (
	"github.com/damianoneill/go-bootstrap/pkg/domain/options"
)

//go:generate mockgen -destination=mocks/mock_metrics.go -package=mocks github.com/damianoneill/go-bootstrap/pkg/domain/metrics Collector,Factory

// Collector handles metrics recording for HTTP requests
type Collector interface {
	// CollectRequestMetrics records metrics for a completed HTTP request
	CollectRequestMetrics(method, path string, status int, duration float64)

	// Close performs any cleanup of the metrics collector
	Close() error
}

// Options configures the behavior of a metrics collector
type Options struct {
	// ServiceName identifies the service in the metrics
	ServiceName string

	// Buckets defines custom histogram buckets for latency metrics
	// If empty, default buckets will be used
	Buckets []float64

	// Labels are additional fixed labels to add to all metrics
	Labels map[string]string

	// Subsystem is an optional name added after the metrics namespace
	// For example: namespace_subsystem_metric_name
	Subsystem string
}

// Option is a function that modifies Options
type Option = options.Option[Options]

// DefaultOptions returns the default metrics options
func DefaultOptions() Options {
	return Options{
		ServiceName: "unknown",
	}
}

// WithDefaults ensures options have proper default values
func WithDefaults(opts *Options) {
	if opts.ServiceName == "" {
		opts.ServiceName = "unknown"
	}
}

// WithServiceName sets the service name that will be included
// in all metrics labels for identification.
func WithServiceName(name string) Option {
	return options.OptionFunc[Options](func(o *Options) error {
		o.ServiceName = name
		return nil
	})
}

// WithBuckets sets custom histogram buckets for latency metrics.
// The buckets should be in ascending order.
func WithBuckets(buckets []float64) Option {
	return options.OptionFunc[Options](func(o *Options) error {
		o.Buckets = buckets
		return nil
	})
}

// WithLabels sets additional labels that will be included
// in all metrics from this collector.
func WithLabels(labels map[string]string) Option {
	return options.OptionFunc[Options](func(o *Options) error {
		o.Labels = labels
		return nil
	})
}

// WithSubsystem sets an optional subsystem name that will be included
// in metric names between the namespace and metric name.
func WithSubsystem(subsystem string) Option {
	return options.OptionFunc[Options](func(o *Options) error {
		o.Subsystem = subsystem
		return nil
	})
}

// Factory creates new metrics collector instances
type Factory interface {
	// NewCollector creates a new metrics collector with the given options
	NewCollector(opts ...Option) (Collector, error)
}
