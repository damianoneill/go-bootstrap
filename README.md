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
        // Service Identity
        ServiceName: "my-service",
        Version:     "1.0.0",

        // Configuration
        ConfigFile:  "config.yaml",
        EnvPrefix:   "MY_SVC",
        EnableConfigViewer: true,  // Enable /internal/config endpoint, runtime config viewer

        // Logging
        LogLevel:    logging.InfoLevel,
        LogFields: logging.Fields{
            "environment": "dev",
            "region":     "us-west",
        },
        EnableLogConfig: true,  // Enable /internal/logging/config, runtime log level configuration

        // HTTP Server
        Port:            8080,
        ReadTimeout:     15 * time.Second,
        WriteTimeout:    15 * time.Second,
        ShutdownTimeout: 15 * time.Second,

        // Observability
        ExcludeFromLogging: []string{"/internal/*", "/metrics"},
        ExcludeFromTracing: []string{"/internal/*", "/metrics"},

        // Tracing
        TracingEndpoint:    "localhost:4317",
        TracingSampleRate:  1.0,
        TracingPropagators: []string{"tracecontext", "baggage"},
    }, deps)

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
    shutdown_timeout: 15s

logging:
  level: info
  fields:
    environment: "dev"
    region: "us-west"

observability:
  exclude_from_logging: ["/health/*", "/metrics"]
  exclude_from_tracing: ["/health/*", "/metrics"]

tracing:
  endpoint: "localhost:4317"
  sample_rate: 1.0
  propagators: ["tracecontext", "baggage"]
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

### HTTP Server Configuration

The library provides flexible HTTP server configuration through two key features:

1. **Expanded Server Options**: The `ServerOptions` struct provides comprehensive server configuration including:
   - Basic settings (port, timeouts)
   - Security options (TLS configuration, certificates)
   - Advanced HTTP tuning (max header size, idle timeout)

```go
svc, err := bootstrap.NewService(bootstrap.Options{
    Server: bootstrap.ServerOptions{
        Port:          8443,
        TLSConfig:     tlsConfig,
        TLSCertFile:   "certs/server.crt",
        TLSKeyFile:    "certs/server.key",
        MaxHeaderSize: 1 << 20,  // 1MB
        IdleTimeout:   60 * time.Second,
    },
})
```

2. **Server Pre-Start Hook**: Applications can customize the `http.Server` before it starts:

```go
Server: bootstrap.ServerOptions{
    PreStart: func(srv *http.Server) error {
        // Customize server before startup
        srv.ErrorLog = log.New(os.Stderr, "[server-error] ", log.LstdFlags)
        return nil
    },
}
```

### Middleware Ordering

The library introduces a structured approach to middleware organization and ordering:

1. **Middleware Categories**: Middleware is now organized into well-defined categories:
   - Core: Fundamental HTTP handling (request ID, recovery, timeouts)
   - Security: Protection (auth, CORS, security headers)
   - Application: Business logic
   - Observability: Logging, metrics, tracing

2. **Configurable Order**: Applications can control middleware execution order:

```go
router, err := bootstrap.NewService(bootstrap.Options{
    Router: domainhttp.RouterOptions{
        MiddlewareOrdering: &domainhttp.MiddlewareOrdering{
            Order: []domainhttp.MiddlewareCategory{
                domainhttp.SecurityMiddleware,   // Run security first
                domainhttp.CoreMiddleware,
                domainhttp.ApplicationMiddleware,
                domainhttp.ObservabilityMiddleware,
            },
            CustomMiddleware: map[domainhttp.MiddlewareCategory][]func(http.Handler) http.Handler{
                domainhttp.SecurityMiddleware: {
                    myAuthMiddleware,
                    myCORSMiddleware,
                },
            },
        },
    },
})
```

The default middleware order prioritizes security and maintains proper observability:

1. Core (fundamental HTTP handling)
2. Security (protection)
3. Application (business logic)
4. Observability (monitoring)

See the [server-customization](./examples/server-customization/main.go) example for a complete demonstration of these features.

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

##  TODO List

- [ ] Security Considerations, what should be handled by the bootstrap vs the application? What should be bootstrap configuration vs application configuration?
  - [ ] Consider CORS.
  - [ ] Consider CSRF.
  - [ ] Consider Rate Limiting.
  - [ ] Consider TLS.
- [ ] Update unit tests.
- [ ] Create integration tests.

##  Security Considerations

The general principle should be:

1. Bootstrap should provide secure defaults and infrastructure
2. Applications should be able to customize security settings for their specific needs
3. Some security features should be mandatory, while others optional

Let's analyze each security consideration:

1. CORS (Cross-Origin Resource Sharing):

- Bootstrap's responsibility:
  - Provide middleware infrastructure for CORS handling
  - Default secure CORS policy (deny all)
  - Standard CORS configuration options structure
- Application's responsibility:
  - Configure allowed origins/methods/headers
  - Handle any special CORS requirements
  - Override CORS behavior for specific routes if needed

2. CSRF (Cross-Site Request Forgery):

- Bootstrap's responsibility:
  - CSRF token generation and validation middleware
  - Secure session handling
  - Standard configuration for token names/headers
- Application's responsibility:
  - Enable/disable CSRF for specific routes
  - Configure token expiration
  - Handle CSRF errors

3. Rate Limiting:

- Bootstrap's responsibility:
  - Rate limiting middleware infrastructure
  - Different rate limiting strategies (token bucket, sliding window etc.)
  - Default limits for common scenarios
- Application's responsibility:
  - Configure rate limits for different endpoints
  - Custom rate limiting rules
  - Handle rate limit exceeded errors

4. TLS:

- Bootstrap's responsibility:
  - TLS configuration infrastructure
  - Secure TLS defaults (modern ciphers, TLS 1.2+)
  - Certificate loading/rotation
- Application's responsibility:
  - Provide certificates
  - Configure specific TLS requirements
  - Handle TLS errors
