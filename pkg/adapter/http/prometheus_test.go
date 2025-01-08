// pkg/adapter/http/prometheus_test.go
package http

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	"github.com/damianoneill/go-bootstrap/pkg/domain/metrics"
)

func TestPrometheusFactory(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	tests := []struct {
		name    string
		options []metrics.Option
		wantErr bool
	}{
		{
			name: "creates collector with default options",
			options: []metrics.Option{
				metrics.WithServiceName("test-service"),
			},
			wantErr: false,
		},
		{
			name: "creates collector with custom options",
			options: []metrics.Option{
				metrics.WithServiceName("custom-service"), // Different name to avoid conflicts
				metrics.WithLabels(map[string]string{
					"environment": "test",
				}),
				metrics.WithBuckets([]float64{0.1, 0.5, 1.0}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset registry for each test case
			prometheus.DefaultRegisterer = prometheus.NewRegistry()

			factory := NewMetricsFactory()
			collector, err := factory.NewCollector(tt.options...)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, collector)

			// Test collector functionality only if creation was successful
			if collector != nil {
				// Record some test metrics
				collector.CollectRequestMetrics("GET", "/test", 200, 0.1)
				time.Sleep(10 * time.Millisecond) // Allow metrics to be processed

				// Clean up
				collector.Close()
			}
		})
	}
}

// TestPrometheusCollectorConcurrency tests the thread safety of the collector
func TestPrometheusCollectorConcurrency(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	factory := NewMetricsFactory()
	collector, err := factory.NewCollector(
		metrics.WithServiceName("concurrent-test"),
	)
	assert.NoError(t, err)
	defer collector.Close()

	// Run concurrent collectors
	const goroutines = 10
	const iterations = 100
	done := make(chan bool)

	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				collector.CollectRequestMetrics("GET", "/test", 200, float64(j)*0.1)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < goroutines; i++ {
		<-done
	}
}

// TestPrometheusCollectorErrors tests error conditions
func TestPrometheusCollectorErrors(t *testing.T) {
	tests := []struct {
		name          string
		setupRegistry func()
		options       []metrics.Option
		wantErrMsg    string
	}{
		{
			name: "fails with invalid service name",
			options: []metrics.Option{
				metrics.WithServiceName(""), // Empty service name should fail
			},
			wantErrMsg: "service name is required",
		},
		{
			name: "fails with invalid bucket values",
			options: []metrics.Option{
				metrics.WithServiceName("test"),
				metrics.WithBuckets([]float64{2.0, 1.0}), // Invalid order
			},
			wantErrMsg: "buckets must be in increasing order",
		},
		{
			name: "fails with duplicate registration",
			setupRegistry: func() {
				hist := prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name: "http_request_duration_seconds",
						Help: "HTTP request duration in seconds",
					},
					[]string{"method", "path", "status"},
				)
				prometheus.MustRegister(hist)
			},
			options: []metrics.Option{
				metrics.WithServiceName("test"),
			},
			wantErrMsg: "registering collector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset registry
			prometheus.DefaultRegisterer = prometheus.NewRegistry()

			if tt.setupRegistry != nil {
				tt.setupRegistry()
			}

			factory := NewMetricsFactory()
			collector, err := factory.NewCollector(tt.options...)

			if tt.wantErrMsg == "" {
				assert.NoError(t, err)
				assert.NotNil(t, collector)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				assert.Nil(t, collector)
			}
		})
	}
}
