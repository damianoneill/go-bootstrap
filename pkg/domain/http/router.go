// Package http provides domain interfaces for HTTP routing and service health probes.
// It builds on chi.Router to provide additional service-level capabilities for
// Cloud Native applications including Kubernetes probe endpoints, observability
// configuration, and service identity management.
package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	"github.com/damianoneill/go-bootstrap/pkg/domain/metrics"
	"github.com/damianoneill/go-bootstrap/pkg/domain/options"
	"github.com/damianoneill/go-bootstrap/pkg/domain/tracing"
)

//go:generate mockgen -destination=mocks/mock_router.go -package=http github.com/damianoneill/go-bootstrap/pkg/domain/http Router,Factory

// MiddlewareCategory represents a classification of middleware
type MiddlewareCategory string

const (
	// CoreMiddleware runs first - handles fundamental HTTP concerns like
	// request IDs, panic recovery, and base timeouts
	CoreMiddleware MiddlewareCategory = "core"

	// SecurityMiddleware runs second - handles authentication, authorization,
	// CORS, CSRF, rate limiting
	SecurityMiddleware MiddlewareCategory = "security"

	// ApplicationMiddleware runs third - handles custom business logic
	// and application-specific concerns
	ApplicationMiddleware MiddlewareCategory = "application"

	// ObservabilityMiddleware runs last - handles logging, metrics,
	// and tracing of the complete request lifecycle
	ObservabilityMiddleware MiddlewareCategory = "observability"
)

// MiddlewareOrdering configures the order of middleware categories
type MiddlewareOrdering struct {
	// Order specifies the sequence of middleware categories
	// If empty, defaults to [Core, Security, Application, Observability]
	Order []MiddlewareCategory
	// CustomMiddleware allows adding middleware to specific categories
	CustomMiddleware map[MiddlewareCategory][]func(http.Handler) http.Handler
}

// requiredCategories defines the middleware categories that must be present
var requiredCategories = map[MiddlewareCategory]struct{}{
	CoreMiddleware:          {},
	SecurityMiddleware:      {},
	ObservabilityMiddleware: {},
}

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

	// MetricsFactory creates a metrics collector for request metrics.
	// If not set, metrics will be disabled
	MetricsFactory metrics.Factory

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

	// MiddlewareOrdering configures middleware ordering
	// If not set, defaults to [Core, Security, Application, Observability]
	MiddlewareOrdering *MiddlewareOrdering
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
		if name == "" {
			return fmt.Errorf("service name cannot be empty")
		}
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

// WithMetricsFactory sets the metrics factory for creating collectors
func WithMetricsFactory(factory metrics.Factory) Option {
	return options.OptionFunc[RouterOptions](func(o *RouterOptions) error {
		o.MetricsFactory = factory
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
		// Validate logging paths
		seen := make(map[string]bool)
		for _, path := range loggingPaths {
			if !strings.HasPrefix(path, "/") {
				return fmt.Errorf("path must start with /: %s", path)
			}
			if seen[path] {
				return fmt.Errorf("duplicate logging path: %s", path)
			}
			seen[path] = true
		}

		// Validate tracing paths
		seen = make(map[string]bool)
		for _, path := range tracingPaths {
			if !strings.HasPrefix(path, "/") {
				return fmt.Errorf("path must start with /: %s", path)
			}
			if seen[path] {
				return fmt.Errorf("duplicate tracing path: %s", path)
			}
			seen[path] = true
		}

		o.ExcludeFromLogging = loggingPaths
		o.ExcludeFromTracing = tracingPaths
		return nil
	})
}

// validateMiddlewareOrdering ensures all required categories are present
func validateMiddlewareOrdering(order []MiddlewareCategory) error {
	if len(order) == 0 {
		return fmt.Errorf("middleware order cannot be empty")
	}

	// Track which categories we've seen
	seen := make(map[MiddlewareCategory]bool)

	// Check for duplicates and build seen set
	for _, category := range order {
		if seen[category] {
			return fmt.Errorf("duplicate middleware category: %s", category)
		}
		seen[category] = true
	}

	// Check that all required categories are present
	var missing []string
	for required := range requiredCategories {
		if !seen[required] {
			missing = append(missing, string(required))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required middleware categories: %s", strings.Join(missing, ", "))
	}

	return nil
}

// WithMiddlewareOrdering sets the order of middleware categories
func WithMiddlewareOrdering(ordering *MiddlewareOrdering) Option {
	return options.OptionFunc[RouterOptions](func(o *RouterOptions) error {
		if ordering == nil {
			return fmt.Errorf("middleware ordering cannot be nil")
		}

		// Validate the ordering includes all required categories
		if err := validateMiddlewareOrdering(ordering.Order); err != nil {
			return fmt.Errorf("invalid middleware ordering: %w", err)
		}

		// Validate custom middleware if provided
		if ordering.CustomMiddleware != nil {
			for category := range ordering.CustomMiddleware {
				// Check that custom middleware is only added to valid categories
				if _, validCategory := requiredCategories[category]; !validCategory &&
					category != ApplicationMiddleware {
					return fmt.Errorf("invalid middleware category for custom middleware: %s", category)
				}
			}
		}

		o.MiddlewareOrdering = ordering
		return nil
	})
}

// Factory creates new router instances with the specified options.
type Factory interface {
	// NewRouter creates a new router with the given options.
	// It will apply service configuration and set up middleware based on the options.
	NewRouter(opts ...Option) (Router, error)
}
