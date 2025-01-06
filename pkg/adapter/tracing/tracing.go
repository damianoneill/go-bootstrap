// pkg/adapter/tracing/otel.go

// Package tracing provides an OpenTelemetry implementation of the tracing domain interfaces
package tracing

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/damianoneill/go-bootstrap/pkg/domain/tracing"
)

// Provider implements the domain Provider interface using OpenTelemetry
type Provider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	enabled  bool
}

// Factory creates OpenTelemetry-based Provider instances
type Factory struct{}

// NewFactory creates a new OpenTelemetry factory
func NewFactory() tracing.Factory {
	return &Factory{}
}

// NewProvider implements Factory.NewProvider
func (f *Factory) NewProvider(opts ...tracing.Option) (tracing.Provider, error) {
	// Initialize default options
	options := &tracing.Options{
		ExporterType: tracing.HTTPExporter,
		SamplingRate: 1.0,
	}

	// Apply options
	for _, opt := range opts {
		if err := opt.ApplyOption(options); err != nil {
			return nil, fmt.Errorf("applying option: %w", err)
		}
	}

	// Validate required fields
	if options.ServiceName == "" {
		return nil, fmt.Errorf("service name is required")
	}

	// Return noop provider if using NoopExporter
	if options.ExporterType == tracing.NoopExporter {
		return &Provider{enabled: false}, nil
	}

	// Create exporter
	exporter, err := f.createExporter(context.Background(), options)
	if err != nil {
		return nil, fmt.Errorf("creating exporter: %w", err)
	}

	// Create resource with service information
	res, err := f.createResource(options)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(f.createSampler(options)),
	)

	// Set as global trace provider
	otel.SetTracerProvider(tp)

	// Configure propagators
	f.setupPropagators(options)

	// Create tracer
	tracer := tp.Tracer(options.ServiceName)

	return &Provider{
		provider: tp,
		tracer:   tracer,
		enabled:  true,
	}, nil
}

// HTTPMiddleware creates an http.Handler that adds tracing
func (f *Factory) HTTPMiddleware(operation string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, operation)
	}
}

// Shutdown implements Provider.Shutdown
func (p *Provider) Shutdown(ctx context.Context) error {
	if !p.enabled || p.provider == nil {
		return nil
	}
	return p.provider.Shutdown(ctx)
}

// IsEnabled implements Provider.IsEnabled
func (p *Provider) IsEnabled() bool {
	return p.enabled
}

// createExporter creates an OTLP exporter based on the configuration
func (f *Factory) createExporter(ctx context.Context, opts *tracing.Options) (sdktrace.SpanExporter, error) {
	switch opts.ExporterType {
	case tracing.HTTPExporter:
		httpOpts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(opts.CollectorEndpoint),
		}

		if opts.Insecure {
			httpOpts = append(httpOpts, otlptracehttp.WithInsecure())
		}

		if len(opts.Headers) > 0 {
			httpOpts = append(httpOpts, otlptracehttp.WithHeaders(opts.Headers))
		}

		return otlptracehttp.New(ctx, httpOpts...)

	case tracing.GRPCExporter:
		grpcOpts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(opts.CollectorEndpoint),
		}

		if opts.Insecure {
			grpcOpts = append(grpcOpts, otlptracegrpc.WithInsecure())
		}

		if len(opts.Headers) > 0 {
			grpcOpts = append(grpcOpts, otlptracegrpc.WithHeaders(opts.Headers))
		}

		return otlptracegrpc.New(ctx, grpcOpts...)

	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", opts.ExporterType)
	}
}

// createResource creates a resource with service information
func (f *Factory) createResource(opts *tracing.Options) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(opts.ServiceName),
			semconv.ServiceVersion(opts.ServiceVersion),
		),
	)
}

// createSampler creates a sampler based on the configuration
func (f *Factory) createSampler(opts *tracing.Options) sdktrace.Sampler {
	if opts.SamplingRate >= 1.0 {
		return sdktrace.AlwaysSample()
	}
	if opts.SamplingRate <= 0.0 {
		return sdktrace.NeverSample()
	}
	return sdktrace.TraceIDRatioBased(opts.SamplingRate)
}

// setupPropagators configures the global propagators
func (f *Factory) setupPropagators(opts *tracing.Options) {
	// Default propagators if none specified
	if len(opts.PropagatorTypes) == 0 {
		opts.PropagatorTypes = []string{
			tracing.PropagatorTraceContext,
			tracing.PropagatorBaggage,
		}
	}

	propagators := make([]propagation.TextMapPropagator, 0, len(opts.PropagatorTypes))
	for _, pType := range opts.PropagatorTypes {
		switch pType {
		case tracing.PropagatorTraceContext:
			propagators = append(propagators, propagation.TraceContext{})
		case tracing.PropagatorBaggage:
			propagators = append(propagators, propagation.Baggage{})
		}
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagators...))
}
