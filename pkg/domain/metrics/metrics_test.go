// pkg/domain/metrics/metrics_test.go
package metrics

import (
	"testing"
)

func defaultMetricsOptions() Options {
	return DefaultOptions()
}

func TestMetricsOptions(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		expected Options
		wantErr  bool
	}{
		{
			name:    "default options",
			options: []Option{},
			expected: Options{
				ServiceName: "unknown",
			},
		},
		{
			name: "set service name",
			options: []Option{
				WithServiceName("test-service"),
			},
			expected: Options{
				ServiceName: "test-service",
			},
		},
		{
			name: "set custom buckets",
			options: []Option{
				WithBuckets([]float64{0.1, 0.5, 1.0, 2.5}),
			},
			expected: Options{
				ServiceName: "unknown",
				Buckets:     []float64{0.1, 0.5, 1.0, 2.5},
			},
		},
		{
			name: "set additional labels",
			options: []Option{
				WithLabels(map[string]string{"env": "production"}),
			},
			expected: Options{
				ServiceName: "unknown",
				Labels:      map[string]string{"env": "production"},
			},
		},
		{
			name: "set subsystem",
			options: []Option{
				WithSubsystem("auth"),
			},
			expected: Options{
				ServiceName: "unknown",
				Subsystem:   "auth",
			},
		},
		{
			name: "set multiple options",
			options: []Option{
				WithServiceName("test-service"),
				WithBuckets([]float64{0.1, 0.5, 1.0}),
				WithLabels(map[string]string{"env": "staging"}),
				WithSubsystem("payment"),
			},
			expected: Options{
				ServiceName: "test-service",
				Buckets:     []float64{0.1, 0.5, 1.0},
				Labels:      map[string]string{"env": "staging"},
				Subsystem:   "payment",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultMetricsOptions()
			for _, opt := range tt.options {
				err := opt.ApplyOption(&opts)
				if (err != nil) != tt.wantErr {
					t.Errorf("ApplyOption() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}

			if opts.ServiceName != tt.expected.ServiceName {
				t.Errorf("ServiceName = %v, want %v", opts.ServiceName, tt.expected.ServiceName)
			}
			if len(opts.Buckets) != len(tt.expected.Buckets) {
				t.Errorf("Buckets length = %v, want %v", len(opts.Buckets), len(tt.expected.Buckets))
			}
			for i, bucket := range tt.expected.Buckets {
				if opts.Buckets[i] != bucket {
					t.Errorf("Buckets[%d] = %v, want %v", i, opts.Buckets[i], bucket)
				}
			}
			if len(opts.Labels) != len(tt.expected.Labels) {
				t.Errorf("Labels length = %v, want %v", len(opts.Labels), len(tt.expected.Labels))
			}
			for k, v := range tt.expected.Labels {
				if opts.Labels[k] != v {
					t.Errorf("Labels[%s] = %v, want %v", k, opts.Labels[k], v)
				}
			}
			if opts.Subsystem != tt.expected.Subsystem {
				t.Errorf("Subsystem = %v, want %v", opts.Subsystem, tt.expected.Subsystem)
			}
		})
	}
}
