// pkg/domain/http/router_test.go
package http_test

import (
	"testing"

	http "github.com/damianoneill/go-bootstrap/pkg/domain/http"
	"github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	"github.com/damianoneill/go-bootstrap/pkg/domain/tracing"
)

func defaultOptions() http.RouterOptions {
	return http.RouterOptions{
		ProbeHandlers: http.DefaultProbeHandlers(),
	}
}

func TestRouterOptions(t *testing.T) {
	// Create test logger and tracer for tests
	testLogger := &mockLogger{}
	testTracer := &mockTracer{}

	tests := []struct {
		name     string
		options  []http.Option
		validate func(*testing.T, http.RouterOptions)
		wantErr  bool
	}{
		{
			name:    "default options",
			options: []http.Option{},
			validate: func(t *testing.T, got http.RouterOptions) {
				if got.ServiceName != "" {
					t.Errorf("ServiceName = %v, want empty", got.ServiceName)
				}
				if got.ServiceVersion != "" {
					t.Errorf("ServiceVersion = %v, want empty", got.ServiceVersion)
				}
				if got.ProbeHandlers == nil {
					t.Error("ProbeHandlers is nil, want default handlers")
				}
			},
		},
		{
			name: "with service info",
			options: []http.Option{
				http.WithService("test-service", "1.0.0"),
			},
			validate: func(t *testing.T, got http.RouterOptions) {
				if got.ServiceName != "test-service" {
					t.Errorf("ServiceName = %v, want test-service", got.ServiceName)
				}
				if got.ServiceVersion != "1.0.0" {
					t.Errorf("ServiceVersion = %v, want 1.0.0", got.ServiceVersion)
				}
			},
		},
		{
			name: "with logger",
			options: []http.Option{
				http.WithLogger(testLogger),
			},
			validate: func(t *testing.T, got http.RouterOptions) {
				if got.Logger == nil {
					t.Error("Logger is nil, want test logger")
				}
			},
		},
		{
			name: "with tracing provider",
			options: []http.Option{
				http.WithTracingProvider(testTracer),
			},
			validate: func(t *testing.T, got http.RouterOptions) {
				if got.TracingProvider == nil {
					t.Error("TracingProvider is nil, want test tracer")
				}
			},
		},
		{
			name: "with probe handlers",
			options: []http.Option{
				http.WithProbeHandlers(http.DefaultProbeHandlers()),
			},
			validate: func(t *testing.T, got http.RouterOptions) {
				if got.ProbeHandlers == nil {
					t.Error("ProbeHandlers is nil, want handlers")
				}
			},
		},
		{
			name: "with observability exclusions",
			options: []http.Option{
				http.WithObservabilityExclusions(
					[]string{"/health"},
					[]string{"/metrics"},
				),
			},
			validate: func(t *testing.T, got http.RouterOptions) {
				if len(got.ExcludeFromLogging) != 1 || got.ExcludeFromLogging[0] != "/health" {
					t.Errorf("ExcludeFromLogging = %v, want [/health]", got.ExcludeFromLogging)
				}
				if len(got.ExcludeFromTracing) != 1 || got.ExcludeFromTracing[0] != "/metrics" {
					t.Errorf("ExcludeFromTracing = %v, want [/metrics]", got.ExcludeFromTracing)
				}
			},
		},
		{
			name: "with multiple options",
			options: []http.Option{
				http.WithService("test-service", "1.0.0"),
				http.WithLogger(testLogger),
				http.WithTracingProvider(testTracer),
				http.WithObservabilityExclusions([]string{"/health"}, []string{"/metrics"}),
			},
			validate: func(t *testing.T, got http.RouterOptions) {
				if got.ServiceName != "test-service" {
					t.Errorf("ServiceName = %v, want test-service", got.ServiceName)
				}
				if got.Logger == nil {
					t.Error("Logger is nil, want test logger")
				}
				if got.TracingProvider == nil {
					t.Error("TracingProvider is nil, want test tracer")
				}
				if len(got.ExcludeFromLogging) != 1 {
					t.Errorf("ExcludeFromLogging length = %v, want 1", len(got.ExcludeFromLogging))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultOptions()

			// Apply options
			for _, opt := range tt.options {
				err := opt.ApplyOption(&opts)
				if (err != nil) != tt.wantErr {
					t.Errorf("ApplyOption() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}

			// Validate results
			tt.validate(t, opts)
		})
	}
}

// Mock implementations for testing
type mockLogger struct {
	logging.Logger
}

type mockTracer struct {
	tracing.Provider
}
