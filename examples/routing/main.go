package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"

	httpadapter "github.com/damianoneill/go-bootstrap/pkg/adapter/http"
	"github.com/damianoneill/go-bootstrap/pkg/adapter/logging"
	"github.com/damianoneill/go-bootstrap/pkg/adapter/tracing"
	domainhttp "github.com/damianoneill/go-bootstrap/pkg/domain/http"
	domainlog "github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	domaintracing "github.com/damianoneill/go-bootstrap/pkg/domain/tracing"
)

var (
	startTime = time.Now()
	logger    domainlog.Logger
)

// printExampleRequests prints curl commands for testing different endpoints
func printExampleRequests() {
	fmt.Println("\n=== Business Endpoints (with tracing & logging) ===")
	fmt.Println("# List users")
	fmt.Println("curl -v http://localhost:8080/api/v1/users")
	fmt.Println("\n# Get user by ID")
	fmt.Println("curl -v http://localhost:8080/api/v1/users/123")
	fmt.Println("\n# Create user")
	fmt.Println(`curl -v -X POST -H "Content-Type: application/json" -d '{"name":"John Doe","email":"john@example.com"}' http://localhost:8080/api/v1/users`)
	fmt.Println("\n# Get user profile (requires auth)")
	fmt.Println(`curl -v -H "Authorization: Bearer token123" http://localhost:8080/api/v1/user/profile`)

	fmt.Println("\n=== Kubernetes Probe Endpoints (excluded from tracing & logging) ===")
	fmt.Println("# Liveness probe")
	fmt.Println("curl -v http://localhost:8080/internal/health")
	fmt.Println("\n# Readiness probe")
	fmt.Println("curl -v http://localhost:8080/internal/ready")
	fmt.Println("\n# Startup probe")
	fmt.Println("curl -v http://localhost:8080/internal/startup")

	fmt.Println("\n=== Observability Endpoints (excluded from tracing & logging) ===")
	fmt.Println("# Prometheus metrics")
	fmt.Println("curl -v http://localhost:8080/metrics")

	fmt.Println("\n=== Testing Trace Context Propagation ===")
	fmt.Println("# Request with trace context")
	fmt.Println("curl -v -H 'traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01' http://localhost:8080/api/v1/users")

	fmt.Println("\n=== OpenTelemetry Collector Configuration ===")
	fmt.Println("# The service is configured to send traces to:")
	fmt.Println("# - Endpoint: localhost:4318")
	fmt.Println("# - Protocol: HTTP")
	fmt.Println("# - Security: Insecure/plaintext")
	fmt.Println("# Make sure your collector is configured to receive traces on this endpoint")

	fmt.Println("\nStarting server...")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()
}

func main() {
	printExampleRequests()

	// Create logger
	logFactory := logging.NewFactory()
	var err error
	logger, err = logFactory.NewLogger(
		domainlog.WithLevel(domainlog.InfoLevel),
		domainlog.WithServiceName("example-service"),
		domainlog.WithFields(domainlog.Fields{
			"environment": "development",
			"version":     "1.0.0",
		}),
	)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting example service")
	router, err := setupRouter()
	if err != nil {
		logger.ErrorWith("Failed to setup router", domainlog.Fields{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// Create server
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.InfoWith("Starting server", domainlog.Fields{
			"address": srv.Addr,
		})
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.ErrorWith("Server error", domainlog.Fields{
				"error": err.Error(),
			})
			os.Exit(1)
		}
	}()

	// Wait for interrupt
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	// Graceful shutdown
	logger.Info("Starting graceful shutdown")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.ErrorWith("Error during shutdown", domainlog.Fields{
			"error": err.Error(),
		})
	}
	logger.Info("Server stopped")
}

func setupRouter() (domainhttp.Router, error) {
	// Create tracer with HTTP exporter
	tracingFactory := tracing.NewFactory()
	tracingProvider, err := tracingFactory.NewProvider(
		domaintracing.WithServiceName("example-service"),
		domaintracing.WithServiceVersion("1.0.0"),
		domaintracing.WithExporterType(domaintracing.HTTPExporter),
		domaintracing.WithCollectorEndpoint("localhost:4318"),
		domaintracing.WithInsecure(true),
		domaintracing.WithSamplingRate(1.0),
		domaintracing.WithHeaders(map[string]string{
			"x-custom-header": "value",
		}),
		domaintracing.WithDefaultPropagators(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating tracer: %w", err)
	}

	// Setup probe handlers with detailed status
	probeHandlers := &domainhttp.ProbeHandlers{
		LivenessCheck: func() domainhttp.ProbeResponse {
			return domainhttp.ProbeResponse{
				Status: "ok",
				Details: map[string]interface{}{
					"version": "1.0.0",
					"uptime":  time.Since(startTime).String(),
					"time":    time.Now().UTC().Format(time.RFC3339),
				},
			}
		},
		ReadinessCheck: func() domainhttp.ProbeResponse {
			// In a real app, check external dependencies
			return domainhttp.ProbeResponse{
				Status: "ok",
				Details: map[string]interface{}{
					"database": "connected",
					"cache":    "ready",
					"outbound": "healthy",
				},
			}
		},
		StartupCheck: func() domainhttp.ProbeResponse {
			return domainhttp.ProbeResponse{
				Status: "ok",
				Details: map[string]interface{}{
					"initialization": "complete",
					"startup_time":   startTime.UTC().Format(time.RFC3339),
				},
			}
		},
	}

	// Create router with options
	factory := httpadapter.NewFactory()
	router, err := factory.NewRouter(
		// Basic service configuration
		domainhttp.WithService("example-service", "1.0.0"),
		// Observability setup
		domainhttp.WithLogger(logger),
		domainhttp.WithTracingProvider(tracingProvider),
		// Probe configuration
		domainhttp.WithProbeHandlers(probeHandlers),
		// Observability exclusions
		domainhttp.WithObservabilityExclusions(
			[]string{"/internal/*", "/metrics"},
			[]string{"/internal/*", "/metrics"},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating router: %w", err)
	}

	// API routes with basepath
	router.Route("/api/v1", func(r chi.Router) {
		// Example business routes
		r.Get("/users", handleGetUsers)
		r.Post("/users", handleCreateUser)
		r.Get("/users/{id}", handleGetUser)

		// Example handler with middleware
		r.Route("/user", func(r chi.Router) {
			r.Use(requireAuth)
			r.Get("/profile", handleUserProfile)
		})
	})

	return router, nil
}

// Example handlers
func handleGetUsers(w http.ResponseWriter, r *http.Request) {
	users := []map[string]interface{}{
		{"id": 1, "name": "Alice"},
		{"id": 2, "name": "Bob"},
	}
	respondJSON(w, http.StatusOK, users)
}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var user map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	respondJSON(w, http.StatusCreated, user)
}

func handleGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user := map[string]interface{}{
		"id":   id,
		"name": "Example User",
	}
	respondJSON(w, http.StatusOK, user)
}

func handleUserProfile(w http.ResponseWriter, r *http.Request) {
	profile := map[string]interface{}{
		"name":  "Example User",
		"email": "user@example.com",
	}
	respondJSON(w, http.StatusOK, profile)
}

// Example middleware
func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authToken := r.Header.Get("Authorization")
		if authToken == "" {
			respondError(w, http.StatusUnauthorized, "Authorization required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.ErrorWith("Failed to encode JSON response", domainlog.Fields{
			"error": err.Error(),
		})
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
