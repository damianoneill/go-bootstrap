# Go Bootstrap

A foundation library for building cloud-native Go applications. This library provides essential building blocks and patterns for creating production-ready microservices with built-in support for configuration, logging, tracing, and health monitoring.

## Features

- 🔧 **Configuration Management**: Environment-based with YAML fallbacks
- 📝 **Structured Logging**: JSON logging with correlation and levels
- 📊 **Metrics**: Prometheus metrics endpoint with request tracking
- 🔍 **Distributed Tracing**: OpenTelemetry integration
- ❤️ **Health Checks**: Kubernetes-compatible probe endpoints
- 🔐 **Security**: TLS support and secure defaults
- 🚦 **Graceful Shutdown**: Clean shutdown with timeout
- 🧪 **Testability**: Dependency injection and interfaces

## Quick Start

1. Install the library:

```bash
go get github.com/damianoneill/go-bootstrap
```

2. Create a new service:

```go
package main

import (
    "github.com/damianoneill/go-bootstrap/pkg/adapter/config"
    "github.com/damianoneill/go-bootstrap/pkg/adapter/http"
    "github.com/damianoneill/go-bootstrap/pkg/adapter/logging"
    "github.com/damianoneill/go-bootstrap/pkg/adapter/tracing"
    "github.com/damianoneill/go-bootstrap/pkg/usecase/bootstrap"
)

func main() {
    // Create dependencies
    deps := bootstrap.Dependencies{
        ConfigFactory:  config.NewFactory(),
        LoggerFactory: logging.NewFactory(),
        RouterFactory: http.NewFactory(),
        TracerFactory: tracing.NewFactory(),
    }

    // Create service
    svc, err := bootstrap.NewService(bootstrap.Options{
        ServiceName: "my-service",
        Version:     "1.0.0",
        EnvPrefix:   "MY_SVC",
    }, deps)
    if err != nil {
        panic(err)
    }

    // Add routes
    router := svc.Router()
    router.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello World!"))
    })

    // Start service
    if err := svc.Run(); err != nil {
        panic(err)
    }
}
```

## Architecture

The library follows Domain-Driven Design and Clean Architecture principles:

```
pkg/
├── domain/      # Core interfaces and types
├── usecase/     # Application use cases and services
├── adapter/     # Interface implementations
└── infrastructure/  # External dependencies
```

### Key Components

- **Domain Layer**: Core interfaces for logging, config, HTTP, and tracing
- **Use Case Layer**: Bootstrap service and business logic
- **Adapter Layer**: Concrete implementations using external libraries
- **Infrastructure Layer**: Third-party integrations (TBD)

## Examples

Complete examples are provided in the `examples/` directory:

- `quickstart/`: Todo service showing core features

Run the quickstart example:

```bash
go run examples/quickstart/main.go
```

## Configuration

Services can be configured via environment variables and/or YAML files:

```yaml
server:
  http:
    port: 8080
    read_timeout: 15s
    write_timeout: 15s

logging:
  level: info

tracing:
  enabled: true
  endpoint: localhost:4317
  sample_rate: 1.0
```

Environment variables override YAML config:

```bash
MY_SVC_PORT=9090 MY_SVC_LOG_LEVEL=debug ./myservice
```

## Health Checks

The library provides Kubernetes-compatible health check endpoints:

- `/internal/health`: Liveness probe
- `/internal/ready`: Readiness probe
- `/internal/startup`: Startup probe

## Metrics

Prometheus metrics are exposed at `/metrics` including:

- Request counts by path and status
- Request duration histograms
- Error counts
- Custom metrics support

## Development

Requirements:

- Go 1.21+
- Make

Setup:

```bash
make dev     # Install tools and generate mocks
make test    # Run tests
make lint    # Run linters
make all     # Full build and test
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This project is inspired by:

- The Go community's best practices
- Clean Architecture principles
- Domain-Driven Design patterns
- Cloud Native Computing Foundation guidelines
