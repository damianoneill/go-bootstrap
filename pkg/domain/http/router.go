// Package http provides domain interfaces for HTTP routing and service health probes.
// It builds on chi.Router to provide additional service-level capabilities for
// Cloud Native applications including Kubernetes probe endpoints, observability
// configuration, and service identity management.
package http

import (
	"github.com/go-chi/chi/v5"

	"github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	"github.com/damianoneill/go-bootstrap/pkg/domain/options"
	"github.com/damianoneill/go-bootstrap/pkg/domain/tracing"
)

// Router extends chi.Router to provide additional service capabilities.
// It inherits all standard HTTP routing functionality from chi while allowing
// implementations to add service-specific features like health probes and
// observability.
type Router interface {
	chi.Router
}

// RouterOptions configures router behavior and service capabilities.
// It focuses on service-level configuration rather than HTTP-specific settings
// which are handled directly by chi.Router.
type RouterOptions struct {
	// ServiceName identifies the service in logs and traces.
	// This should be a concise, lowercase identifier like "ordersvc".
	ServiceName string

	// ServiceVersion identifies the version of the service.
	// This should follow semantic versioning like "1.2.3".
	ServiceVersion string

	// Logger provides structured logging capabilities.
	// If not set, logging will be disabled.
	Logger logging.Logger

	// TracingProvider enables distributed tracing.
	// If not set, tracing will be disabled.
	TracingProvider tracing.Provider

	// ProbeHandlers configures Kubernetes probe endpoints.
	// If not set, default handlers returning healthy will be used.
	ProbeHandlers *ProbeHandlers

	// ExcludeFromLogging lists paths that should not be logged.
	// Typically used for high-volume health check endpoints to reduce noise.
	// Paths should be exact matches like "/internal/health".
	ExcludeFromLogging []string

	// ExcludeFromTracing lists paths that should not be traced.
	// Typically used for health check endpoints or internal routes.
	// Paths should be exact matches like "/internal/ready".
	ExcludeFromTracing []string
}

// Option is a function that modifies RouterOptions following the
// functional options pattern.
type Option = options.Option[RouterOptions]

// WithService sets the service name and version for identification in
// logs, traces, and other observability outputs.
//
// The name should be a concise, lowercase identifier like "ordersvc".
// The version should follow semantic versioning like "1.2.3".
func WithService(name, version string) Option {
	return options.OptionFunc[RouterOptions](func(o *RouterOptions) error {
		o.ServiceName = name
		o.ServiceVersion = version
		return nil
	})
}

// WithLogger sets the logger for request logging.
// If not set, logging will be disabled.
func WithLogger(logger logging.Logger) Option {
	return options.OptionFunc[RouterOptions](func(o *RouterOptions) error {
		o.Logger = logger
		return nil
	})
}

// WithTracingProvider sets the tracing provider for distributed tracing.
// If not set, tracing will be disabled.
func WithTracingProvider(provider tracing.Provider) Option {
	return options.OptionFunc[RouterOptions](func(o *RouterOptions) error {
		o.TracingProvider = provider
		return nil
	})
}

// WithProbeHandlers sets custom probe handler functions for
// Kubernetes liveness, readiness, and startup probes.
func WithProbeHandlers(handlers *ProbeHandlers) Option {
	return options.OptionFunc[RouterOptions](func(o *RouterOptions) error {
		o.ProbeHandlers = handlers
		return nil
	})
}

// WithObservabilityExclusions sets paths to exclude from both
// logging and tracing. This is typically used for health check
// endpoints to reduce observability noise.
//
// Paths should be exact matches like "/internal/health".
func WithObservabilityExclusions(loggingPaths []string, tracingPaths []string) Option {
	return options.OptionFunc[RouterOptions](func(o *RouterOptions) error {
		o.ExcludeFromLogging = loggingPaths
		o.ExcludeFromTracing = tracingPaths
		return nil
	})
}

// WithLoggingExclusions sets paths to exclude from request logging.
// This is typically used for high-volume endpoints to reduce log noise.
//
// Paths should be exact matches like "/internal/health".
func WithLoggingExclusions(paths []string) Option {
	return options.OptionFunc[RouterOptions](func(o *RouterOptions) error {
		o.ExcludeFromLogging = paths
		return nil
	})
}

// WithTracingExclusions sets paths to exclude from tracing.
// This is typically used for health check endpoints or internal routes.
//
// Paths should be exact matches like "/internal/ready".
func WithTracingExclusions(paths []string) Option {
	return options.OptionFunc[RouterOptions](func(o *RouterOptions) error {
		o.ExcludeFromTracing = paths
		return nil
	})
}

// Factory creates new router instances with the specified options.
type Factory interface {
	// NewRouter creates a new router with the given options.
	// It will apply service configuration and set up middleware based on the options.
	NewRouter(opts ...Option) (Router, error)
}
