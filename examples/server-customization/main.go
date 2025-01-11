// examples/server-customization/main.go
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/damianoneill/go-bootstrap/pkg/adapter/config"
	httpadapter "github.com/damianoneill/go-bootstrap/pkg/adapter/http"
	"github.com/damianoneill/go-bootstrap/pkg/adapter/logging"
	"github.com/damianoneill/go-bootstrap/pkg/adapter/metrics"
	"github.com/damianoneill/go-bootstrap/pkg/adapter/tracing"
	domainhttp "github.com/damianoneill/go-bootstrap/pkg/domain/http"
	domainlog "github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	"github.com/damianoneill/go-bootstrap/pkg/usecase/bootstrap"
)

func main() {
	// Create bootstrap dependencies
	deps := bootstrap.Dependencies{
		ConfigFactory:  config.NewFactory(),
		LoggerFactory:  logging.NewFactory(),
		RouterFactory:  httpadapter.NewFactory(),
		TracerFactory:  tracing.NewFactory(),
		MetricsFactory: metrics.NewMetricsFactory(),
	}

	// Initialize logger early for startup logging
	logger, err := logging.NewFactory().NewLogger(
		domainlog.WithLevel(domainlog.InfoLevel),
		domainlog.WithServiceName("server-example"),
		domainlog.WithFields(domainlog.Fields{
			"environment": "dev",
		}),
	)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// Calculate absolute paths for certificates
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get working directory: %v\n", err)
		os.Exit(1)
	}
	certFile := filepath.Join(pwd, "examples/server-customization/certs/server.crt")
	keyFile := filepath.Join(pwd, "examples/server-customization/certs/server.key")

	// Create TLS configuration with modern security settings
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			// TLS 1.3 ciphers
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			// TLS 1.2 ciphers
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		NextProtos: []string{"h2", "http/1.1"}, // Enable HTTP/2
	}

	// Create service with advanced server configuration
	svc, err := bootstrap.NewService(bootstrap.Options{
		ServiceName: "server-example",
		Version:     "1.0.0",
		ConfigFile:  "examples/server-customization/config.yaml",
		EnvPrefix:   "SRV_EXAMPLE",

		// Enhanced logging
		LogLevel: domainlog.InfoLevel,
		LogFields: domainlog.Fields{
			"environment": "dev",
			"component":   "server-example",
		},
		EnableLogConfig: true,

		// Advanced server configuration
		Server: bootstrap.ServerOptions{
			Port:            9443,
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    15 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			MaxHeaderSize:   1 << 20, // 1MB

			// TLS Configuration
			TLSConfig:   tlsConfig,
			TLSCertFile: certFile,
			TLSKeyFile:  keyFile,

			// Pre-start hook for server customization
			PreStart: func(srv *http.Server) error {
				// Log that PreStart is executing
				fmt.Printf("[pre-start] Configuring server...\n")

				// Add custom base context with multiple values
				srv.BaseContext = func(l net.Listener) context.Context {
					ctx := context.Background()
					ctx = context.WithValue(ctx, "server_start", time.Now())
					ctx = context.WithValue(ctx, "server_addr", l.Addr().String())
					ctx = context.WithValue(ctx, "server_network", l.Addr().Network())
					return ctx
				}

				// Custom error logger with more visible format
				srv.ErrorLog = log.New(os.Stderr, "\n[SERVER-ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)

				return nil
			},
		},

		// Add middleware ordering configuration
		Router: domainhttp.RouterOptions{
			MiddlewareOrdering: &domainhttp.MiddlewareOrdering{
				Order: []domainhttp.MiddlewareCategory{
					domainhttp.SecurityMiddleware, // Security first
					domainhttp.CoreMiddleware,
					domainhttp.ApplicationMiddleware,
					domainhttp.ObservabilityMiddleware,
				},
				CustomMiddleware: map[domainhttp.MiddlewareCategory][]func(http.Handler) http.Handler{
					domainhttp.SecurityMiddleware: {
						// Move security headers middleware here
						func(next http.Handler) http.Handler {
							return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								w.Header().Set("X-Content-Type-Options", "nosniff")
								w.Header().Set("X-Frame-Options", "DENY")
								w.Header().Set("X-XSS-Protection", "1; mode=block")
								next.ServeHTTP(w, r)
							})
						},
					},
					domainhttp.ApplicationMiddleware: {
						// Move rate limiting middleware here
						middleware.ThrottleBacklog(10, 50, time.Second*10),
					},
				},
			},
		},

		// Observability configuration
		ExcludeFromLogging: []string{"/internal/*", "/metrics"},
		ExcludeFromTracing: []string{"/internal/*", "/metrics"},

		TracingEndpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		TracingSampleRate:  1.0,
		TracingPropagators: []string{"tracecontext", "baggage"},
	}, deps, nil)

	if err != nil {
		logger.ErrorWith("Failed to create service", domainlog.Fields{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// Add custom routes
	router := svc.Router()
	router.Route("/api/v1", func(r chi.Router) {
		// Middleware now configured through MiddlewareOrdering above
		r.Get("/secure", handleSecureEndpoint(svc))
		r.Get("/tls-debug", handleTLSDebug(svc))
	})

	// Start service
	errChan := make(chan error, 1)
	go func() {
		if err := svc.Start(); err != nil {
			errChan <- fmt.Errorf("failed to start service: %w", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.ErrorWith("Service error", domainlog.Fields{
			"error": err.Error(),
		})
		os.Exit(1)
	case sig := <-sigChan:
		logger.InfoWith("Received shutdown signal", domainlog.Fields{
			"signal": sig.String(),
		})
	}

	// Graceful shutdown
	if err := svc.Shutdown(context.Background()); err != nil {
		logger.ErrorWith("Shutdown error", domainlog.Fields{
			"error": err.Error(),
		})
		os.Exit(1)
	}
}

// Helper function to convert TLS version to string
func tlsVersionToString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (%d)", version)
	}
}

// handleSecureEndpoint demonstrates TLS information access
func handleSecureEndpoint(svc *bootstrap.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := svc.Logger()

		// Get server start time from context
		if startTime, ok := r.Context().Value("server_start").(time.Time); ok {
			logger.InfoWith("Server uptime", domainlog.Fields{
				"uptime": time.Since(startTime).String(),
			})
		}

		// Get TLS connection state
		tlsState := r.TLS
		response := map[string]interface{}{
			"message":          "Secure endpoint accessed",
			"tls":              true,
			"tls_version":      tlsVersionToString(tlsState.Version),
			"cipher_suite":     tls.CipherSuiteName(tlsState.CipherSuite),
			"negotiated_proto": tlsState.NegotiatedProtocol,
			"server_name":      tlsState.ServerName,
			"time":             time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// handleTLSDebug provides detailed TLS connection information
func handleTLSDebug(svc *bootstrap.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tlsState := r.TLS
		if tlsState == nil {
			http.Error(w, "Not a TLS connection", http.StatusBadRequest)
			return
		}

		// Collect detailed TLS information
		debug := map[string]interface{}{
			"tls_version":        tlsVersionToString(tlsState.Version),
			"cipher_suite":       tls.CipherSuiteName(tlsState.CipherSuite),
			"negotiated_proto":   tlsState.NegotiatedProtocol,
			"server_name":        tlsState.ServerName,
			"handshake_complete": tlsState.HandshakeComplete,
			"mutual":             tlsState.HandshakeComplete && len(tlsState.PeerCertificates) > 0,
			"client_addr":        r.RemoteAddr,
			"connection": map[string]interface{}{
				"reused":   tlsState.DidResume,
				"protocol": r.Proto,
				"alpn":     tlsState.NegotiatedProtocol,
				"http2":    r.ProtoMajor == 2,
			},
		}
		// Add certificate info if server certificates are present
		if len(tlsState.ServerName) > 0 {
			debug["certificate"] = map[string]interface{}{
				"server_name": tlsState.ServerName,
			}
		}

		if len(tlsState.PeerCertificates) > 0 {
			cert := tlsState.PeerCertificates[0]
			debug["peer_certificate"] = map[string]interface{}{
				"issued_to":  cert.Subject.CommonName,
				"issued_by":  cert.Issuer.CommonName,
				"valid_from": cert.NotBefore.Format(time.RFC3339),
				"valid_to":   cert.NotAfter.Format(time.RFC3339),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(debug)
	}
}
