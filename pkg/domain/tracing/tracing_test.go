// pkg/domain/tracing/tracing_test.go
package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithServiceName(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		wantName string
	}{
		{
			name:     "sets service name",
			value:    "test-service",
			wantName: "test-service",
		},
		{
			name:     "empty service name",
			value:    "",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := WithServiceName(tt.value)
			opts := &Options{}
			err := opt.ApplyOption(opts)
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, opts.ServiceName)
		})
	}
}

func TestWithSamplingRate(t *testing.T) {
	tests := []struct {
		name    string
		rate    float64
		wantErr bool
	}{
		{
			name:    "valid rate - 0.0",
			rate:    0.0,
			wantErr: false,
		},
		{
			name:    "valid rate - 0.5",
			rate:    0.5,
			wantErr: false,
		},
		{
			name:    "valid rate - 1.0",
			rate:    1.0,
			wantErr: false,
		},
		{
			name:    "invalid rate - negative",
			rate:    -0.1,
			wantErr: true,
		},
		{
			name:    "invalid rate - above 1",
			rate:    1.1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := WithSamplingRate(tt.rate)
			opts := &Options{}
			err := opt.ApplyOption(opts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.rate, opts.SamplingRate)
		})
	}
}

func TestWithPropagatorTypes(t *testing.T) {
	tests := []struct {
		name  string
		types []string
		want  []string
	}{
		{
			name:  "standard propagators",
			types: []string{PropagatorTraceContext, PropagatorBaggage},
			want:  []string{PropagatorTraceContext, PropagatorBaggage},
		},
		{
			name:  "all propagators",
			types: []string{PropagatorTraceContext, PropagatorBaggage, PropagatorB3, PropagatorJaeger},
			want:  []string{PropagatorTraceContext, PropagatorBaggage, PropagatorB3, PropagatorJaeger},
		},
		{
			name:  "empty propagators",
			types: []string{},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := WithPropagatorTypes(tt.types)
			opts := &Options{}
			err := opt.ApplyOption(opts)
			require.NoError(t, err)
			assert.Equal(t, tt.want, opts.PropagatorTypes)
		})
	}
}

func TestWithExporterType(t *testing.T) {
	tests := []struct {
		name         string
		exporterType ExporterType
		want         ExporterType
	}{
		{
			name:         "HTTP exporter",
			exporterType: HTTPExporter,
			want:         HTTPExporter,
		},
		{
			name:         "GRPC exporter",
			exporterType: GRPCExporter,
			want:         GRPCExporter,
		},
		{
			name:         "Noop exporter",
			exporterType: NoopExporter,
			want:         NoopExporter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := WithExporterType(tt.exporterType)
			opts := &Options{}
			err := opt.ApplyOption(opts)
			require.NoError(t, err)
			assert.Equal(t, tt.want, opts.ExporterType)
		})
	}
}

func TestWithDefaultPropagators(t *testing.T) {
	opt := WithDefaultPropagators()
	opts := &Options{}
	err := opt.ApplyOption(opts)
	require.NoError(t, err)

	expected := []string{PropagatorTraceContext, PropagatorBaggage}
	assert.Equal(t, expected, opts.PropagatorTypes)
}

func TestWithAllPropagators(t *testing.T) {
	opt := WithAllPropagators()
	opts := &Options{}
	err := opt.ApplyOption(opts)
	require.NoError(t, err)

	expected := []string{PropagatorTraceContext, PropagatorBaggage, PropagatorB3, PropagatorJaeger}
	assert.Equal(t, expected, opts.PropagatorTypes)
}

func TestWithInsecure(t *testing.T) {
	tests := []struct {
		name     string
		insecure bool
		want     bool
	}{
		{
			name:     "enable insecure",
			insecure: true,
			want:     true,
		},
		{
			name:     "disable insecure",
			insecure: false,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := WithInsecure(tt.insecure)
			opts := &Options{}
			err := opt.ApplyOption(opts)
			require.NoError(t, err)
			assert.Equal(t, tt.want, opts.Insecure)
		})
	}
}

func TestWithHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    map[string]string
	}{
		{
			name: "with headers",
			headers: map[string]string{
				"Authorization": "Bearer token",
				"Custom":        "Value",
			},
			want: map[string]string{
				"Authorization": "Bearer token",
				"Custom":        "Value",
			},
		},
		{
			name:    "empty headers",
			headers: map[string]string{},
			want:    map[string]string{},
		},
		{
			name:    "nil headers",
			headers: nil,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := WithHeaders(tt.headers)
			opts := &Options{}
			err := opt.ApplyOption(opts)
			require.NoError(t, err)
			assert.Equal(t, tt.want, opts.Headers)
		})
	}
}
