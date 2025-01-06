// pkg/domain/tracing/tracing.go

// Package tracing defines the core tracing interfaces and configuration options
// for distributed tracing support using OpenTelemetry.
package tracing

import (
	"context"
	"fmt"
	"net/http"

	"github.com/damianoneill/go-bootstrap/pkg/domain/options"
)

// Provider manages the creation and configuration of tracers.
// It handles the lifecycle of trace collection and export.
type Provider interface {
	// Shutdown cleanly stops the tracer provider and ensures all spans are exported.
	// The context controls how long to wait for export completion.
	Shutdown(ctx context.Context) error

	// IsEnabled returns whether tracing is currently active.
	// This can be used to conditionally add spans or attributes.
	IsEnabled() bool
}

// ExporterType defines the type of OpenTelemetry exporter to use.
type ExporterType string

const (
	// HTTPExporter configures the provider to use OTLP over HTTP protocol
	HTTPExporter ExporterType = "http"

	// GRPCExporter configures the provider to use OTLP over gRPC protocol
	GRPCExporter ExporterType = "grpc"

	// NoopExporter disables tracing by using a no-operation exporter
	NoopExporter ExporterType = "noop"
)

// PropagatorType defines standard trace context propagation formats
const (
	// PropagatorTraceContext enables W3C Trace Context propagation
	PropagatorTraceContext = "tracecontext"

	// PropagatorBaggage enables W3C Baggage propagation
	PropagatorBaggage = "baggage"

	// PropagatorB3 enables B3 propagation (both single and multi-header)
	PropagatorB3 = "b3"

	// PropagatorJaeger enables Jaeger propagation
	PropagatorJaeger = "jaeger"
)

// Options configures the tracer provider behavior.
// All fields are optional and have reasonable defaults.
type Options struct {
	// ServiceName identifies the service in the trace data
	ServiceName string

	// ServiceVersion identifies the version of the service
	ServiceVersion string

	// CollectorEndpoint is the URL of the OpenTelemetry collector
	// For GRPC, default is "localhost:4317"
	// For HTTP, default is "localhost:4318"
	CollectorEndpoint string

	// ExporterType defines the protocol for sending trace data
	// Default is HTTPExporter
	ExporterType ExporterType

	// Headers are added to OTLP requests (e.g., for authentication)
	// Example: {"api-key": "secret"}
	Headers map[string]string

	// Insecure disables TLS for the exporter connection
	// Default is false (TLS enabled)
	Insecure bool

	// PropagatorTypes defines which context propagation formats to support
	// Default is ["tracecontext", "baggage"]
	PropagatorTypes []string

	// SamplingRate sets the probability of trace sampling (0.0-1.0)
	// Default is 1.0 (sample everything)
	SamplingRate float64
}

// Option is a function that modifies Options
type Option = options.Option[Options]

// Factory creates configured Provider instances
type Factory interface {
	// NewProvider creates a new Provider with the given options
	NewProvider(opts ...Option) (Provider, error)

	// HTTPMiddleware creates an http.Handler middleware that adds tracing
	// The operation parameter sets the name of the created spans
	HTTPMiddleware(operation string) func(http.Handler) http.Handler
}

// WithServiceName sets the service name for span attribution
func WithServiceName(name string) Option {
	return options.OptionFunc[Options](func(o *Options) error {
		o.ServiceName = name
		return nil
	})
}

// WithServiceVersion sets the service version for span attribution
func WithServiceVersion(version string) Option {
	return options.OptionFunc[Options](func(o *Options) error {
		o.ServiceVersion = version
		return nil
	})
}

// WithCollectorEndpoint sets the OpenTelemetry collector endpoint
func WithCollectorEndpoint(endpoint string) Option {
	return options.OptionFunc[Options](func(o *Options) error {
		o.CollectorEndpoint = endpoint
		return nil
	})
}

// WithExporterType sets the type of exporter to use
func WithExporterType(exporterType ExporterType) Option {
	return options.OptionFunc[Options](func(o *Options) error {
		o.ExporterType = exporterType
		return nil
	})
}

// WithHeaders sets headers to be included in OTLP requests
func WithHeaders(headers map[string]string) Option {
	return options.OptionFunc[Options](func(o *Options) error {
		o.Headers = headers
		return nil
	})
}

// WithInsecure sets whether to disable TLS
func WithInsecure(insecure bool) Option {
	return options.OptionFunc[Options](func(o *Options) error {
		o.Insecure = insecure
		return nil
	})
}

// WithPropagatorTypes sets the context propagation formats to support
func WithPropagatorTypes(types []string) Option {
	return options.OptionFunc[Options](func(o *Options) error {
		o.PropagatorTypes = types
		return nil
	})
}

// WithSamplingRate sets the trace sampling probability
// rate must be between 0.0 and 1.0
func WithSamplingRate(rate float64) Option {
	return options.OptionFunc[Options](func(o *Options) error {
		if rate < 0.0 || rate > 1.0 {
			return fmt.Errorf("sampling rate must be between 0.0 and 1.0")
		}
		o.SamplingRate = rate
		return nil
	})
}

// WithDefaultPropagators configures standard W3C propagation
func WithDefaultPropagators() Option {
	return WithPropagatorTypes([]string{
		PropagatorTraceContext,
		PropagatorBaggage,
	})
}

// WithAllPropagators enables all supported propagation formats
func WithAllPropagators() Option {
	return WithPropagatorTypes([]string{
		PropagatorTraceContext,
		PropagatorBaggage,
		PropagatorB3,
		PropagatorJaeger,
	})
}
