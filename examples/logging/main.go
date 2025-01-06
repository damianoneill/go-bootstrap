package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/damianoneill/go-bootstrap/pkg/adapter/logging"
	domainlog "github.com/damianoneill/go-bootstrap/pkg/domain/logging"
)

func initTracer() (*sdktrace.TracerProvider, error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("example-service"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

func main() {
	// Initialize tracer
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Create tracer
	tracer := tp.Tracer("example-service")

	// Create factory
	factory := logging.NewFactory()

	fmt.Println("\n=== Example 1: Basic logger with domain options ===")
	logger, err := factory.NewLogger(
		domainlog.WithLevel(domainlog.InfoLevel),
		domainlog.WithServiceName("example-service"),
		domainlog.WithFields(domainlog.Fields{
			"environment": "development",
			"version":     "1.0.0",
		}),
	)
	if err != nil {
		panic(err)
	}

	logger.Info("Starting service")

	fmt.Println("\n=== Example 2: Logger with both domain and Zap options ===")
	zapLogger, err := factory.NewLoggerWithOptions(
		[]domainlog.Option{
			domainlog.WithLevel(domainlog.DebugLevel),
			domainlog.WithServiceName("example-service-dev"),
		},
		[]logging.ZapOption{
			logging.WithDevelopment(true),
		},
	)
	if err != nil {
		panic(err)
	}

	zapLogger.DebugWith("Configuration loaded", domainlog.Fields{
		"config_path": "/etc/config.yaml",
		"debug_mode":  true,
	})

	fmt.Println("\n=== Example 3: Using With() for request context ===")
	requestLogger := logger.With(domainlog.Fields{
		"request_id": "123e4567-e89b-12d3-a456-426614174000",
		"user_id":    "user123",
	})

	requestLogger.InfoWith("Processing request", domainlog.Fields{
		"path":   "/api/v1/users",
		"method": "GET",
	})

	fmt.Println("\n=== Example 4: Using WithContext for trace context ===")
	// Create root context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create span
	ctx, span := tracer.Start(ctx, "example-operation")
	defer span.End()

	traceLogger := logger.WithContext(ctx)
	traceLogger.InfoWith("Executing traced operation", domainlog.Fields{
		"operation": "database_query",
		"table":     "users",
	})

	fmt.Println("\n=== Example 5: Different log levels ===")
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.WarnWith("Warning message", domainlog.Fields{
		"attention_level": "medium",
	})
	logger.ErrorWith("Error occurred", domainlog.Fields{
		"error_code": 500,
		"error_msg":  "database connection failed",
	})

	fmt.Println("\n=== Example 6: Changing log levels dynamically ===")
	zapLogger.SetLevel(domainlog.WarnLevel)
	zapLogger.Debug("This won't be logged")
	zapLogger.Warn("This will be logged")

	fmt.Println("\n=== Example 7: Complex tracing scenario ===")
	// Create new trace context
	ctx, rootSpan := tracer.Start(ctx, "root-operation")
	defer rootSpan.End()

	rootLogger := logger.WithContext(ctx)
	rootLogger.Info("Starting complex operation")

	// Create child span
	ctx, childSpan := tracer.Start(ctx, "child-operation")
	defer childSpan.End()

	childLogger := logger.WithContext(ctx)
	childLogger.InfoWith("Processing child operation", domainlog.Fields{
		"operation_type": "database-query",
		"query_params": domainlog.Fields{
			"table": "users",
			"limit": 10,
		},
	})

	// Handle shutdown signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	select {
	case <-signalChan:
		logger.Info("Received shutdown signal")
	case <-ctx.Done():
		logger.Info("Context canceled")
	case <-time.After(2 * time.Second):
		logger.Info("Example completed")
	}
}
