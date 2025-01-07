// pkg/adapter/tracing/otel_test.go

package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/damianoneill/go-bootstrap/pkg/domain/tracing"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	assert.NotNil(t, factory)
	assert.IsType(t, &Factory{}, factory)
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name    string
		opts    []tracing.Option
		wantErr bool
	}{
		{
			name: "valid configuration with http exporter",
			opts: []tracing.Option{
				tracing.WithServiceName("test-service"),
				tracing.WithExporterType(tracing.HTTPExporter),
				tracing.WithCollectorEndpoint("localhost:4318"),
			},
			wantErr: false,
		},
		{
			name: "valid configuration with grpc exporter",
			opts: []tracing.Option{
				tracing.WithServiceName("test-service"),
				tracing.WithExporterType(tracing.GRPCExporter),
				tracing.WithCollectorEndpoint("localhost:4317"),
				tracing.WithInsecure(true),
			},
			wantErr: false,
		},
		{
			name: "noop exporter",
			opts: []tracing.Option{
				tracing.WithServiceName("test-service"),
				tracing.WithExporterType(tracing.NoopExporter),
			},
			wantErr: false,
		},
		{
			name: "missing service name",
			opts: []tracing.Option{
				tracing.WithExporterType(tracing.HTTPExporter),
			},
			wantErr: true,
		},
		{
			name: "invalid exporter type",
			opts: []tracing.Option{
				tracing.WithServiceName("test-service"),
				tracing.WithExporterType("invalid"),
			},
			wantErr: true,
		},
		{
			name: "with sampling rate",
			opts: []tracing.Option{
				tracing.WithServiceName("test-service"),
				tracing.WithSamplingRate(0.5),
			},
			wantErr: false,
		},
		{
			name: "with headers",
			opts: []tracing.Option{
				tracing.WithServiceName("test-service"),
				tracing.WithHeaders(map[string]string{
					"Authorization": "Bearer token",
				}),
			},
			wantErr: false,
		},
		{
			name: "with propagators",
			opts: []tracing.Option{
				tracing.WithServiceName("test-service"),
				tracing.WithDefaultPropagators(),
			},
			wantErr: false,
		},
		{
			name: "with all propagators",
			opts: []tracing.Option{
				tracing.WithServiceName("test-service"),
				tracing.WithAllPropagators(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory()
			provider, err := factory.NewProvider(tt.opts...)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, provider)

			// Test shutdown
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			err = provider.Shutdown(ctx)
			assert.NoError(t, err)
		})
	}
}

func TestProvider_Shutdown(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		wantErr  bool
	}{
		{
			name: "disabled provider",
			provider: &Provider{
				enabled: false,
			},
			wantErr: false,
		},
		{
			name: "nil provider",
			provider: &Provider{
				enabled:  true,
				provider: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.provider.Shutdown(ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProvider_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		want     bool
	}{
		{
			name: "enabled provider",
			provider: &Provider{
				enabled: true,
			},
			want: true,
		},
		{
			name: "disabled provider",
			provider: &Provider{
				enabled: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.provider.IsEnabled())
		})
	}
}

func TestFactory_HTTPMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		operation      string
		expectedStatus int
		requestHeaders map[string]string
	}{
		{
			name:           "basic handler",
			operation:      "test-operation",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "with trace headers",
			operation:      "test-operation",
			expectedStatus: http.StatusOK,
			requestHeaders: map[string]string{
				"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create provider with noop exporter for testing
			factory := NewFactory()
			provider, err := factory.NewProvider(
				tracing.WithServiceName("test-service"),
				tracing.WithExporterType(tracing.NoopExporter),
				tracing.WithDefaultPropagators(),
			)
			require.NoError(t, err)
			defer func() {
				if err := provider.Shutdown(context.Background()); err != nil {
					t.Errorf("failed to shutdown provider: %v", err)
				}
			}()

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.expectedStatus)
			})

			// Create middleware
			middleware := factory.HTTPMiddleware(tt.operation)
			require.NotNil(t, middleware)

			// Create traced handler
			tracedHandler := middleware(handler)

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			for k, v := range tt.requestHeaders {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()

			// Execute request
			tracedHandler.ServeHTTP(rec, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}
