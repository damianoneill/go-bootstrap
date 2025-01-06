// Package http provides a Chi-based implementation of the HTTP routing domain interfaces.
package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	domainhttp "github.com/damianoneill/go-bootstrap/pkg/domain/http"
	"github.com/damianoneill/go-bootstrap/pkg/domain/logging"
)

// ChiRouter wraps chi.Router with additional service capabilities
type ChiRouter struct {
	chi.Router // Embed chi.Router to inherit HTTP routing
	opts       domainhttp.RouterOptions
	metrics    *metrics
	matcher    *defaultMatcher
}

// Factory creates Chi-based router instances
type Factory struct{}

// NewFactory creates a new Chi router factory
func NewFactory() *Factory {
	return &Factory{}
}

// NewRouter implements the domain Factory interface
func (f *Factory) NewRouter(opts ...domainhttp.Option) (domainhttp.Router, error) {
	// Initialize options with defaults
	options := domainhttp.RouterOptions{
		ProbeHandlers: domainhttp.DefaultProbeHandlers(),
	}

	// Apply provided options
	for _, opt := range opts {
		if err := opt.ApplyOption(&options); err != nil {
			return nil, fmt.Errorf("applying router option: %w", err)
		}
	}

	// Validate required options
	if options.ServiceName == "" {
		return nil, fmt.Errorf("service name is required")
	}

	// Create router instance
	r := &ChiRouter{
		Router:  chi.NewRouter(),
		opts:    options,
		metrics: newMetrics(options.ServiceName),
		matcher: newMatcher(),
	}

	// Configure router
	if err := r.configure(); err != nil {
		return nil, fmt.Errorf("configuring router: %w", err)
	}

	return r, nil
}

// configure sets up middleware and routes
func (r *ChiRouter) configure() error {
	// Add base middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Add tracing first so it can set up the context
	if r.opts.TracingProvider != nil {
		r.Use(r.tracingMiddleware())
	}

	// Then add logging which will use the trace context
	if r.opts.Logger != nil {
		r.Use(r.loggingMiddleware())
	}

	// Add metrics collection
	r.Use(r.metricsMiddleware())

	// Configure probes
	r.Mount("/internal", r.probeRoutes())

	// Add metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	return nil
}

// probeRoutes creates a router with probe endpoints
func (r *ChiRouter) probeRoutes() chi.Router {
	probes := chi.NewRouter()

	probes.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		r.handleProbe(w, r.opts.ProbeHandlers.LivenessCheck())
	})

	probes.Get("/ready", func(w http.ResponseWriter, _ *http.Request) {
		r.handleProbe(w, r.opts.ProbeHandlers.ReadinessCheck())
	})

	probes.Get("/startup", func(w http.ResponseWriter, _ *http.Request) {
		r.handleProbe(w, r.opts.ProbeHandlers.StartupCheck())
	})

	return probes
}

// handleProbe writes a probe response with appropriate status code
func (r *ChiRouter) handleProbe(w http.ResponseWriter, resp domainhttp.ProbeResponse) {
	w.Header().Set("Content-Type", "application/json")

	if resp.Status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		if r.opts.Logger != nil {
			r.opts.Logger.ErrorWith("Failed to encode probe response", logging.Fields{
				"error": err.Error(),
			})
		}
		w.WriteHeader(http.StatusInternalServerError)
		// Try to write a simple error response
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"}); err != nil {
			// If we can't even encode the error response, just give up
			return
		}
	}
}

// loggingMiddleware creates a middleware for request logging
func (r *ChiRouter) loggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Skip excluded paths
			if r.matcher.Matches(req.URL.Path, r.opts.ExcludeFromLogging) {
				next.ServeHTTP(w, req)
				return
			}

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)

			defer func() {
				// Use WithContext to include trace information
				contextLogger := r.opts.Logger.WithContext(req.Context())

				contextLogger.InfoWith("HTTP Request", logging.Fields{
					"method":     req.Method,
					"path":       req.URL.Path,
					"status":     ww.Status(),
					"duration":   time.Since(start).String(),
					"size":       ww.BytesWritten(),
					"request_id": middleware.GetReqID(req.Context()),
				})
			}()

			next.ServeHTTP(ww, req)
		})
	}
}

// tracingMiddleware creates a middleware for request tracing
func (r *ChiRouter) tracingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if r.matcher.Matches(req.URL.Path, r.opts.ExcludeFromTracing) {
				next.ServeHTTP(w, req)
				return
			}

			// Create operation name from request
			operation := fmt.Sprintf("%s %s", req.Method, req.URL.Path)

			// Use otelhttp with proper operation name and capture router options
			opts := r.opts // capture options for use in formatter
			handler := otelhttp.NewHandler(
				next,
				operation,
				otelhttp.WithSpanNameFormatter(func(operation string, req *http.Request) string {
					return fmt.Sprintf("%s.http %s", opts.ServiceName, operation)
				}),
			)
			handler.ServeHTTP(w, req)
		})
	}
}

// metricsMiddleware creates a middleware for metrics collection
func (r *ChiRouter) metricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Skip excluded paths
			if r.matcher.Matches(req.URL.Path, r.opts.ExcludeFromLogging) {
				next.ServeHTTP(w, req)
				return
			}

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			next.ServeHTTP(ww, req)

			// Record metrics
			path := r.normalizePath(req)
			duration := time.Since(start).Seconds()
			r.metrics.collectMetrics(req.Method, path, ww.Status(), duration)
		})
	}
}

// normalizePath helps prevent high cardinality in metrics by normalizing dynamic path segments
func (r *ChiRouter) normalizePath(req *http.Request) string {
	if rctx := chi.RouteContext(req.Context()); rctx != nil && rctx.RoutePattern() != "" {
		return rctx.RoutePattern()
	}
	return req.URL.Path
}
