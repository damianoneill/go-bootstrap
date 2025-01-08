// pkg/domain/http/router_test.go
package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mocklog "github.com/damianoneill/go-bootstrap/pkg/domain/logging/mocks"
	mocktracing "github.com/damianoneill/go-bootstrap/pkg/domain/tracing/mocks"
)

func defaultOptions() RouterOptions {
	return RouterOptions{
		ProbeHandlers: DefaultProbeHandlers(),
	}
}

func TestRouterOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLogger := mocklog.NewMockLogger(ctrl)
	testTracer := mocktracing.NewMockProvider(ctrl)

	tests := []struct {
		name     string
		options  []Option
		validate func(*testing.T, RouterOptions)
		wantErr  bool
	}{
		{
			name:    "default options",
			options: []Option{},
			validate: func(t *testing.T, got RouterOptions) {
				assert.Empty(t, got.ServiceName)
				assert.Empty(t, got.ServiceVersion)
				assert.NotNil(t, got.ProbeHandlers)
			},
		},
		{
			name: "with service info",
			options: []Option{
				WithService("test-service", "1.0.0"),
			},
			validate: func(t *testing.T, got RouterOptions) {
				assert.Equal(t, "test-service", got.ServiceName)
				assert.Equal(t, "1.0.0", got.ServiceVersion)
			},
		},
		{
			name: "with logger",
			options: []Option{
				WithLogger(testLogger),
			},
			validate: func(t *testing.T, got RouterOptions) {
				assert.NotNil(t, got.Logger)
			},
		},
		{
			name: "with tracing provider",
			options: []Option{
				WithTracingProvider(testTracer),
			},
			validate: func(t *testing.T, got RouterOptions) {
				assert.NotNil(t, got.TracingProvider)
			},
		},
		{
			name: "with probe handlers",
			options: []Option{
				WithProbeHandlers(DefaultProbeHandlers()),
			},
			validate: func(t *testing.T, got RouterOptions) {
				assert.NotNil(t, got.ProbeHandlers)
			},
		},
		{
			name: "with multiple options",
			options: []Option{
				WithService("test-service", "1.0.0"),
				WithLogger(testLogger),
				WithTracingProvider(testTracer),
				WithObservabilityExclusions([]string{"/health"}, []string{"/metrics"}),
			},
			validate: func(t *testing.T, got RouterOptions) {
				assert.Equal(t, "test-service", got.ServiceName)
				assert.NotNil(t, got.Logger)
				assert.NotNil(t, got.TracingProvider)
				assert.Equal(t, []string{"/health"}, got.ExcludeFromLogging)
				assert.Equal(t, []string{"/metrics"}, got.ExcludeFromTracing)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultOptions()

			// Apply options
			for _, opt := range tt.options {
				err := opt.ApplyOption(&opts)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
			}

			// Validate results
			tt.validate(t, opts)
		})
	}
}

func TestRouterOptionsValidation(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		wantErr string
	}{
		{
			name: "empty service name",
			options: []Option{
				WithService("", "1.0.0"),
			},
			wantErr: "service name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := RouterOptions{}
			var err error
			for _, opt := range tt.options {
				if err = opt.ApplyOption(&opts); err != nil {
					break
				}
			}
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestObservabilityExclusions(t *testing.T) {
	tests := []struct {
		name         string
		loggingPaths []string
		tracingPaths []string
		validate     func(*testing.T, RouterOptions)
		wantErr      string
	}{
		{
			name:         "valid single path exclusions",
			loggingPaths: []string{"/health"},
			tracingPaths: []string{"/metrics"},
			validate: func(t *testing.T, got RouterOptions) {
				assert.Equal(t, []string{"/health"}, got.ExcludeFromLogging)
				assert.Equal(t, []string{"/metrics"}, got.ExcludeFromTracing)
			},
		},
		{
			name:         "valid multiple paths",
			loggingPaths: []string{"/health", "/metrics"},
			tracingPaths: []string{"/ready", "/live"},
			validate: func(t *testing.T, got RouterOptions) {
				assert.Equal(t, []string{"/health", "/metrics"}, got.ExcludeFromLogging)
				assert.Equal(t, []string{"/ready", "/live"}, got.ExcludeFromTracing)
			},
		},
		{
			name:         "path in both lists is valid",
			loggingPaths: []string{"/health"},
			tracingPaths: []string{"/health"},
			validate: func(t *testing.T, got RouterOptions) {
				assert.Equal(t, []string{"/health"}, got.ExcludeFromLogging)
				assert.Equal(t, []string{"/health"}, got.ExcludeFromTracing)
			},
		},
		{
			name:         "empty paths are valid",
			loggingPaths: []string{},
			tracingPaths: []string{},
			validate: func(t *testing.T, got RouterOptions) {
				assert.Empty(t, got.ExcludeFromLogging)
				assert.Empty(t, got.ExcludeFromTracing)
			},
		},
		{
			name: "nil paths are valid",
			validate: func(t *testing.T, got RouterOptions) {
				assert.Empty(t, got.ExcludeFromLogging)
				assert.Empty(t, got.ExcludeFromTracing)
			},
		},
		{
			name:         "duplicate in logging paths",
			loggingPaths: []string{"/health", "/health"},
			wantErr:      "duplicate logging path: /health",
		},
		{
			name:         "duplicate in tracing paths",
			tracingPaths: []string{"/metrics", "/metrics"},
			wantErr:      "duplicate tracing path: /metrics",
		},
		{
			name:         "invalid logging path format",
			loggingPaths: []string{"health"},
			wantErr:      "path must start with /: health",
		},
		{
			name:         "invalid tracing path format",
			tracingPaths: []string{"metrics"},
			wantErr:      "path must start with /: metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := RouterOptions{}
			err := WithObservabilityExclusions(tt.loggingPaths, tt.tracingPaths).ApplyOption(&opts)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			assert.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, opts)
			}
		})
	}
}
