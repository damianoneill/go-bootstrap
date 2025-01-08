// Package http provides a Chi-based implementation of the HTTP routing domain interfaces.
package http

import (
	"context"
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
	"github.com/damianoneill/go-bootstrap/pkg/domain/metrics"
)

// Router implements the domain Router interface using Chi
type Router struct {
	chi.Router                   // Embed chi.Router for HTTP routing
	opts       RouterOptions     // Configuration options
	metrics    metrics.Collector // Metrics collector for instrumentation
	matcher    *defaultMatcher   // Path matcher for exclusions
}

// RouterOptions contains the effective configuration for the router
type RouterOptions struct {
	domainhttp.RouterOptions
	middleware []func(http.Handler) http.Handler // List of configured middleware
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

	// Create metrics collector if metrics factory provided
	var metricsCollector metrics.Collector
	if options.MetricsFactory != nil {
		collector, err := options.MetricsFactory.NewCollector(
			metrics.WithServiceName(options.ServiceName),
			metrics.WithLabels(map[string]string{
				"version": options.ServiceVersion,
			}),
		)
		if err != nil {
			return nil, fmt.Errorf("creating metrics collector: %w", err)
		}
		metricsCollector = collector
	}

	// Create and configure router
	router, err := newRouter(options, metricsCollector)
	if err != nil {
		return nil, fmt.Errorf("creating router: %w", err)
	}

	return router, nil
}

// newRouter creates a new configured Router instance
func newRouter(opts domainhttp.RouterOptions, collector metrics.Collector) (*Router, error) {
	r := &Router{
		Router:  chi.NewRouter(),
		opts:    RouterOptions{RouterOptions: opts},
		metrics: collector,
		matcher: newMatcher(),
	}

	// Create and configure middleware
	if err := r.configureMiddleware(); err != nil {
		return nil, fmt.Errorf("configuring middleware: %w", err)
	}

	// Configure routes
	if err := r.configureRoutes(); err != nil {
		return nil, fmt.Errorf("configuring routes: %w", err)
	}

	return r, nil
}

// configureMiddleware sets up all middleware in the correct order
func (r *Router) configureMiddleware() error {
	// Add base middleware
	r.opts.middleware = append(r.opts.middleware,
		middleware.RequestID,
		middleware.RealIP,
		middleware.Recoverer,
		middleware.Timeout(30*time.Second),
	)

	// Add tracing if configured
	if r.opts.TracingProvider != nil {
		r.opts.middleware = append(r.opts.middleware, r.tracingMiddleware())
	}

	// Add logging if configured
	if r.opts.Logger != nil {
		r.opts.middleware = append(r.opts.middleware, r.loggingMiddleware())
	}

	// Add metrics collection
	if r.metrics != nil {
		r.opts.middleware = append(r.opts.middleware, r.metricsMiddleware())
	}

	// Apply all middleware
	for _, mw := range r.opts.middleware {
		r.Use(mw)
	}

	return nil
}

// configureRoutes sets up all routes including probe and metrics endpoints
func (r *Router) configureRoutes() error {
	// Configure internal routes
	internal := chi.NewRouter()

	// Health probe routes
	internal.Get("/health", r.probeHandler(r.opts.ProbeHandlers.LivenessCheck))
	internal.Get("/ready", r.probeHandler(r.opts.ProbeHandlers.ReadinessCheck))
	internal.Get("/startup", r.probeHandler(r.opts.ProbeHandlers.StartupCheck))

	// Mount internal routes
	r.Mount("/internal", internal)

	// Add metrics endpoint if collector configured
	if r.metrics != nil {
		r.Handle("/metrics", promhttp.Handler())
	}

	return nil
}

// probeHandler creates a handler for probe endpoints
func (r *Router) probeHandler(check domainhttp.ProbeCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		resp := check()
		if err := r.writeProbeResponse(w, resp); err != nil {
			if r.opts.Logger != nil {
				r.opts.Logger.ErrorWith("Failed to write probe response", logging.Fields{
					"error": err.Error(),
				})
			}
		}
	}
}

// writeProbeResponse writes a probe response with appropriate status code
func (r *Router) writeProbeResponse(w http.ResponseWriter, resp domainhttp.ProbeResponse) error {
	w.Header().Set("Content-Type", "application/json")

	if resp.Status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	return json.NewEncoder(w).Encode(resp)
}

// loggingMiddleware creates a middleware for request logging
func (r *Router) loggingMiddleware() func(http.Handler) http.Handler {
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
func (r *Router) tracingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if r.matcher.Matches(req.URL.Path, r.opts.ExcludeFromTracing) {
				next.ServeHTTP(w, req)
				return
			}

			// Create operation name from request
			operation := fmt.Sprintf("%s %s", req.Method, req.URL.Path)

			// Use otelhttp with proper operation name
			opts := r.opts // capture options for formatter
			handler := otelhttp.NewHandler(
				next,
				operation,
				otelhttp.WithSpanNameFormatter(func(operation string, _ *http.Request) string {
					return fmt.Sprintf("%s.http %s", opts.RouterOptions.ServiceName, operation)
				}),
			)
			handler.ServeHTTP(w, req)
		})
	}
}

// metricsMiddleware creates a middleware for collecting request metrics
func (r *Router) metricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Skip if no metrics collector or excluded path
			if r.metrics == nil || r.matcher.Matches(req.URL.Path, r.opts.ExcludeFromLogging) {
				next.ServeHTTP(w, req)
				return
			}

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			next.ServeHTTP(ww, req)

			// Record metrics
			duration := time.Since(start).Seconds()
			path := r.normalizePath(req)
			r.metrics.CollectRequestMetrics(req.Method, path, ww.Status(), duration)
		})
	}
}

// normalizePath returns a normalized path for metrics collection
func (r *Router) normalizePath(req *http.Request) string {
	if rctx := chi.RouteContext(req.Context()); rctx != nil && rctx.RoutePattern() != "" {
		return rctx.RoutePattern()
	}
	return req.URL.Path
}

// Close handles cleanup of router resources
func (r *Router) Close(ctx context.Context) error {
	var errs []error

	// Close metrics collector if configured
	if r.metrics != nil {
		if err := r.metrics.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing metrics collector: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("closing router: %v", errs)
	}

	return nil
}
