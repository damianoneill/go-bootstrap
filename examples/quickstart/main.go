// examples/quickstart/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/damianoneill/go-bootstrap/pkg/adapter/config"
	httpadapter "github.com/damianoneill/go-bootstrap/pkg/adapter/http"
	"github.com/damianoneill/go-bootstrap/pkg/adapter/logging"
	"github.com/damianoneill/go-bootstrap/pkg/adapter/metrics"
	"github.com/damianoneill/go-bootstrap/pkg/adapter/tracing"
	domainhttp "github.com/damianoneill/go-bootstrap/pkg/domain/http"
	domainlog "github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	"github.com/damianoneill/go-bootstrap/pkg/usecase/bootstrap"
)

type Todo struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

// Custom probe handler structure to track application metrics
type applicationProbe struct {
	startTime  time.Time
	todosCount int
	logger     domainlog.Logger
}

// Creates a new applicationProbe with custom metrics
func newApplicationProbe(logger domainlog.Logger) *applicationProbe {
	return &applicationProbe{
		startTime: time.Now(),
		logger:    logger,
	}
}

func (p *applicationProbe) createProbeHandlers() *domainhttp.ProbeHandlers {
	return &domainhttp.ProbeHandlers{
		LivenessCheck: func() domainhttp.ProbeResponse {
			return domainhttp.ProbeResponse{
				Status: "ok",
				Details: map[string]interface{}{
					"goroutines": runtime.NumGoroutine(),
					"uptime":     time.Since(p.startTime).String(),
					"memory": map[string]interface{}{
						"alloc":   runtime.MemStats{}.Alloc,
						"objects": runtime.MemStats{}.HeapObjects,
					},
				},
			}
		},
		ReadinessCheck: func() domainhttp.ProbeResponse {
			// Custom readiness logic that checks todos count
			status := "ok"
			if p.todosCount > 1000 { // Example threshold
				status = "degraded"
			}
			return domainhttp.ProbeResponse{
				Status: status,
				Details: map[string]interface{}{
					"todos_count": p.todosCount,
					"started_at":  p.startTime.Format(time.RFC3339),
				},
			}
		},
		StartupCheck: func() domainhttp.ProbeResponse {
			return domainhttp.ProbeResponse{
				Status: "ok",
				Details: map[string]interface{}{
					"started_at": p.startTime.Format(time.RFC3339),
				},
			}
		},
	}
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

	// Initialize logger early for startup logging
	logger, err := logging.NewFactory().NewLogger(
		domainlog.WithLevel(domainlog.InfoLevel),
		domainlog.WithServiceName("todo-service"),
		domainlog.WithFields(domainlog.Fields{
			"environment": "dev",
		}),
	)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// Create probe handler
	probe := newApplicationProbe(logger)

	// Create bootstrap service
	svc, err := bootstrap.NewService(bootstrap.Options{
		ServiceName:        "todo-service",
		Version:            "1.0.0",
		ConfigFile:         "examples/quickstart/config.yaml",
		EnvPrefix:          "TODO_SVC",
		EnableConfigViewer: true,

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

		// Custom probe handlers, if not set default handlers will be used
		ProbeHandlers: probe.createProbeHandlers(),

		// Tracing configuration
		TracingEndpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		TracingSampleRate:  1.0,
		TracingPropagators: []string{"tracecontext", "baggage"},
	}, deps, nil) // No hooks needed for production use

	if err != nil {
		logger.ErrorWith("Failed to create service", domainlog.Fields{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// Get router and add routes
	router := svc.Router()
	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/todos", handleGetTodos(svc, probe))
		r.Post("/todos", handleCreateTodo(svc, probe))
		r.Get("/todos/{id}", handleGetTodo(svc))
		r.Put("/todos/{id}", handleUpdateTodo(svc))
		r.Delete("/todos/{id}", handleDeleteTodo(svc, probe))
	})

	// Print API documentation using logger
	printAPIDoc(logger)

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
		logger.ErrorWith("Service error", domainlog.Fields{
			"error": err.Error(),
		})
		os.Exit(1)
	case sig := <-sigChan:
		logger.InfoWith("Received shutdown signal", domainlog.Fields{
			"signal": sig.String(),
		})
	}

	// Graceful shutdown
	if err := svc.Shutdown(context.Background()); err != nil {
		logger.ErrorWith("Shutdown error", domainlog.Fields{
			"error": err.Error(),
		})
		os.Exit(1)
	}
}

// In-memory store for demo purposes
var todos = make(map[string]*Todo)

func handleGetTodos(svc *bootstrap.Service, probe *applicationProbe) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := svc.Logger()

		todoList := make([]*Todo, 0, len(todos))
		for _, todo := range todos {
			todoList = append(todoList, todo)
		}

		probe.todosCount = len(todoList)
		logger.InfoWith("Fetching all todos", domainlog.Fields{
			"count": len(todoList),
		})
		respondJSON(w, http.StatusOK, todoList)
	}
}

// Modified create handler to update probe metrics
func handleCreateTodo(svc *bootstrap.Service, probe *applicationProbe) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := svc.Logger()

		var todo Todo
		if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
			logger.ErrorWith("Invalid request body", domainlog.Fields{
				"error": err.Error(),
			})
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		todo.ID = fmt.Sprintf("todo_%d", time.Now().UnixNano())
		todo.CreatedAt = time.Now()

		todos[todo.ID] = &todo
		probe.todosCount = len(todos)

		logger.InfoWith("Created todo", domainlog.Fields{
			"id":          todo.ID,
			"total_todos": len(todos),
		})
		respondJSON(w, http.StatusCreated, todo)
	}
}

// Modified delete handler to update probe metrics
func handleDeleteTodo(svc *bootstrap.Service, probe *applicationProbe) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := svc.Logger()
		id := chi.URLParam(r, "id")

		if _, exists := todos[id]; !exists {
			logger.WarnWith("Todo not found", domainlog.Fields{
				"id": id,
			})
			respondError(w, http.StatusNotFound, "Todo not found")
			return
		}

		delete(todos, id)
		probe.todosCount = len(todos)

		logger.InfoWith("Deleted todo", domainlog.Fields{
			"id":          id,
			"total_todos": len(todos),
		})
		respondJSON(w, http.StatusNoContent, nil)
	}
}

// Helper function to print API documentation using logger
func printAPIDoc(logger domainlog.Logger) {
	logger.Info("=== Todo Service API ===")

	/**
	  Example API usage with curl:

	  List all todos:
	  curl -X GET http://localhost:8080/api/v1/todos

	  Get a specific todo:
	  curl -X GET http://localhost:8080/api/v1/todos/todo_1234567890

	  Create a new todo:
	  curl -X POST http://localhost:8080/api/v1/todos \
	    -H "Content-Type: application/json" \
	    -d '{"title": "Learn Go", "completed": false}'

	  Update a todo:
	  curl -X PUT http://localhost:8080/api/v1/todos/todo_1234567890 \
	    -H "Content-Type: application/json" \
	    -d '{"title": "Learn Go", "completed": true}'

	  Delete a todo:
	  curl -X DELETE http://localhost:8080/api/v1/todos/todo_1234567890

	  Check health:
	  curl -X GET http://localhost:8080/internal/health

	  Check readiness:
	  curl -X GET http://localhost:8080/internal/ready

	  Check startup status:
	  curl -X GET http://localhost:8080/internal/startup

	  Get metrics:
	  curl -X GET http://localhost:8080/metrics

	  Example responses:
	  Success: {"success":true,"data":{"id":"todo_1234567890","title":"Learn Go","completed":false}}
	  Error: {"success":false,"error":"Todo not found"}
	*/
	logger.InfoWith("Available Endpoints", domainlog.Fields{
		"list_todos":    "GET    /api/v1/todos",
		"get_todo":      "GET    /api/v1/todos/{id}",
		"create_todo":   "POST   /api/v1/todos",
		"update_todo":   "PUT    /api/v1/todos/{id}",
		"delete_todo":   "DELETE /api/v1/todos/{id}",
		"health_check":  "GET    /internal/health",
		"ready_check":   "GET    /internal/ready",
		"startup_check": "GET    /internal/startup",
		"metrics":       "GET    /metrics",
	})

	logger.InfoWith("Environment Variables", domainlog.Fields{
		"TODO_SVC_PORT":               "HTTP port (default: 8080)",
		"TODO_SVC_LOG_LEVEL":          "Log level (default: info)",
		"OTEL_EXPORTER_OTLP_ENDPOINT": "OpenTelemetry collector endpoint",
	})

	logger.Info("Starting server...")
}

// Get todo by ID handler
func handleGetTodo(svc *bootstrap.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := svc.Logger()
		id := chi.URLParam(r, "id")

		todo, exists := todos[id]
		if !exists {
			logger.WarnWith("Todo not found", domainlog.Fields{
				"id": id,
			})
			respondError(w, http.StatusNotFound, "Todo not found")
			return
		}

		logger.InfoWith("Fetched todo", domainlog.Fields{
			"id":        id,
			"completed": todo.Completed,
		})
		respondJSON(w, http.StatusOK, todo)
	}
}

// Update todo handler
func handleUpdateTodo(svc *bootstrap.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := svc.Logger()
		id := chi.URLParam(r, "id")

		existing, exists := todos[id]
		if !exists {
			logger.WarnWith("Todo not found", domainlog.Fields{
				"id": id,
			})
			respondError(w, http.StatusNotFound, "Todo not found")
			return
		}

		var updates Todo
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			logger.ErrorWith("Invalid request body", domainlog.Fields{
				"error": err.Error(),
				"id":    id,
			})
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Track changes for logging
		changes := domainlog.Fields{
			"id": id,
		}

		if existing.Title != updates.Title {
			changes["old_title"] = existing.Title
			changes["new_title"] = updates.Title
			existing.Title = updates.Title
		}

		if existing.Completed != updates.Completed {
			changes["old_completed"] = existing.Completed
			changes["new_completed"] = updates.Completed
			existing.Completed = updates.Completed
		}

		logger.InfoWith("Updated todo", changes)
		respondJSON(w, http.StatusOK, existing)
	}
}

// APIResponse wraps all API responses in a consistent structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// respondJSON sends a JSON response with proper headers
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	response := APIResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Note: At this point we can't reliably send another JSON response
		// since we may have already written response headers
		w.Write([]byte(`{"success":false,"error":"Failed to encode response"}`))
	}
}

// respondError sends an error response with proper headers
func respondError(w http.ResponseWriter, status int, message string) {
	response := APIResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Best effort to write the error response
	json.NewEncoder(w).Encode(response)
}
