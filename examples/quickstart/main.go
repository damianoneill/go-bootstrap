// examples/quickstart/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/damianoneill/go-bootstrap/pkg/adapter/config"
	httpadapter "github.com/damianoneill/go-bootstrap/pkg/adapter/http"
	"github.com/damianoneill/go-bootstrap/pkg/adapter/logging"
	"github.com/damianoneill/go-bootstrap/pkg/adapter/metrics"
	"github.com/damianoneill/go-bootstrap/pkg/adapter/tracing"
	domainlog "github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	"github.com/damianoneill/go-bootstrap/pkg/usecase/bootstrap"
)

type Todo struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	// Create bootstrap dependencies
	deps := bootstrap.Dependencies{
		ConfigFactory:  config.NewFactory(),
		LoggerFactory:  logging.NewFactory(),
		RouterFactory:  httpadapter.NewFactory(),
		TracerFactory:  tracing.NewFactory(),
		MetricsFactory: metrics.NewMetricsFactory(),
	}

	// Create bootstrap service
	svc, err := bootstrap.NewService(bootstrap.Options{
		ServiceName: "todo-service",
		Version:     "1.0.0",
		ConfigFile:  "examples/quickstart/config.yaml",
		EnvPrefix:   "TODO_SVC",

		// Enhanced logging
		LogLevel: domainlog.InfoLevel,
		LogFields: domainlog.Fields{
			"environment": "dev",
			"component":   "todo-service",
		},
		EnableLogConfig: true,

		// Server timeouts
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		ShutdownTimeout: 15 * time.Second,

		// Observability exclusions
		ExcludeFromLogging: []string{"/internal/*", "/metrics"},
		ExcludeFromTracing: []string{"/internal/*", "/metrics"},

		// Tracing configuration
		TracingEndpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		TracingSampleRate:  1.0,
		TracingPropagators: []string{"tracecontext", "baggage"},
	}, deps, nil) // No hooks needed for production use

	if err != nil {
		fmt.Printf("Failed to create service: %v\n", err)
		os.Exit(1)
	}

	// Get router and add routes
	router := svc.Router()
	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/todos", handleGetTodos(svc))
		r.Post("/todos", handleCreateTodo(svc))
		r.Get("/todos/{id}", handleGetTodo(svc))
		r.Put("/todos/{id}", handleUpdateTodo(svc))
		r.Delete("/todos/{id}", handleDeleteTodo(svc))
	})

	// Print API documentation
	printAPIDoc()

	// Start service
	errChan := make(chan error, 1)
	go func() {
		if err := svc.Start(); err != nil {
			errChan <- fmt.Errorf("failed to start service: %w", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		fmt.Printf("Service error: %v\n", err)
		os.Exit(1)
	case sig := <-sigChan:
		fmt.Printf("Received signal: %v\n", sig)
	}

	// Graceful shutdown
	if err := svc.Shutdown(context.Background()); err != nil {
		fmt.Printf("Shutdown error: %v\n", err)
		os.Exit(1)
	}
}

// In-memory store for demo purposes
var todos = make(map[string]*Todo)

func handleGetTodos(svc *bootstrap.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := svc.Logger()

		// Initialize with empty slice instead of nil
		todoList := make([]*Todo, 0)
		for _, todo := range todos {
			todoList = append(todoList, todo)
		}

		logger.Info("Fetching all todos")
		respondJSON(w, http.StatusOK, todoList)
	}
}

func handleGetTodo(svc *bootstrap.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := svc.Logger()
		id := chi.URLParam(r, "id")

		todo, exists := todos[id]
		if !exists {
			logger.WarnWith("Todo not found", map[string]interface{}{
				"id": id,
			})
			respondError(w, http.StatusNotFound, "Todo not found")
			return
		}

		logger.InfoWith("Fetching todo", map[string]interface{}{
			"id": id,
		})
		respondJSON(w, http.StatusOK, todo)
	}
}

func handleCreateTodo(svc *bootstrap.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := svc.Logger()

		var todo Todo
		if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
			logger.ErrorWith("Invalid request body", map[string]interface{}{
				"error": err.Error(),
			})
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Generate ID and set creation time
		todo.ID = fmt.Sprintf("todo_%d", time.Now().UnixNano())
		todo.CreatedAt = time.Now()

		todos[todo.ID] = &todo

		logger.InfoWith("Created todo", map[string]interface{}{
			"id": todo.ID,
		})
		respondJSON(w, http.StatusCreated, todo)
	}
}

func handleUpdateTodo(svc *bootstrap.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := svc.Logger()
		id := chi.URLParam(r, "id")

		existing, exists := todos[id]
		if !exists {
			logger.WarnWith("Todo not found", map[string]interface{}{
				"id": id,
			})
			respondError(w, http.StatusNotFound, "Todo not found")
			return
		}

		var updates Todo
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			logger.ErrorWith("Invalid request body", map[string]interface{}{
				"error": err.Error(),
			})
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Update fields
		existing.Title = updates.Title
		existing.Completed = updates.Completed

		logger.InfoWith("Updated todo", map[string]interface{}{
			"id": id,
		})
		respondJSON(w, http.StatusOK, existing)
	}
}

func handleDeleteTodo(svc *bootstrap.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := svc.Logger()
		id := chi.URLParam(r, "id")

		if _, exists := todos[id]; !exists {
			logger.WarnWith("Todo not found", map[string]interface{}{
				"id": id,
			})
			respondError(w, http.StatusNotFound, "Todo not found")
			return
		}

		delete(todos, id)

		logger.InfoWith("Deleted todo", map[string]interface{}{
			"id": id,
		})
		respondJSON(w, http.StatusNoContent, nil)
	}
}

// APIResponse wraps all API responses in a consistent structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	response := APIResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	response := APIResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func printAPIDoc() {
	fmt.Println("\n=== Todo Service API ===")
	fmt.Println("\n# List all todos")
	fmt.Println("curl -X GET http://localhost:8080/api/v1/todos")

	fmt.Println("\n# Get todo by ID")
	fmt.Println("curl -X GET http://localhost:8080/api/v1/todos/{id}")

	fmt.Println("\n# Create todo")
	fmt.Println(`curl -X POST http://localhost:8080/api/v1/todos \
	-H "Content-Type: application/json" \
	-d '{"title":"Learn Go","completed":false}'`)

	fmt.Println("\n# Update todo")
	fmt.Println(`curl -X PUT http://localhost:8080/api/v1/todos/{id} \
	-H "Content-Type: application/json" \
	-d '{"title":"Learn Go","completed":true}'`)

	fmt.Println("\n# Delete todo")
	fmt.Println("curl -X DELETE http://localhost:8080/api/v1/todos/{id}")

	fmt.Println("\n=== Health & Metrics ===")
	fmt.Println("# Liveness probe")
	fmt.Println("curl http://localhost:8080/internal/health")

	fmt.Println("\n# Readiness probe")
	fmt.Println("curl http://localhost:8080/internal/ready")

	fmt.Println("\n# Metrics")
	fmt.Println("curl http://localhost:8080/metrics")

	fmt.Println("\n=== Environment Variables ===")
	fmt.Println("TODO_SVC_PORT        - HTTP port (default: 8080)")
	fmt.Println("TODO_SVC_LOG_LEVEL   - Log level (default: info)")
	fmt.Println("OTEL_EXPORTER_OTLP_ENDPOINT - OpenTelemetry collector endpoint (optional)")

	fmt.Println("\nStarting server...")
}
