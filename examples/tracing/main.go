package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/damianoneill/go-bootstrap/pkg/adapter/tracing"
	domaintracing "github.com/damianoneill/go-bootstrap/pkg/domain/tracing"
)

func main() {
	// Create tracing factory
	factory := tracing.NewFactory()

	// Create provider with options
	provider, err := factory.NewProvider(
		domaintracing.WithServiceName("example-service"),
		domaintracing.WithServiceVersion("1.0.0"),
		domaintracing.WithCollectorEndpoint("localhost:4317"),
		domaintracing.WithExporterType(domaintracing.GRPCExporter),
		domaintracing.WithSamplingRate(1.0),
		domaintracing.WithInsecure(true),
	)
	if err != nil {
		log.Fatalf("Failed to create tracer provider: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down provider: %v", err)
		}
	}()

	// Create router and add traced endpoints
	mux := http.NewServeMux()

	// Add traced endpoints
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!")
	})

	// Create server with traced handler
	tracedHandler := factory.HTTPMiddleware("http-server")(mux)
	srv := &http.Server{
		Addr:    ":8080",
		Handler: tracedHandler,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on :8080")
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	// Graceful shutdown
	log.Printf("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
