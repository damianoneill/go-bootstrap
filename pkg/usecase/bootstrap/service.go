// pkg/usecase/bootstrap/service.go

package bootstrap

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	domainconfig "github.com/damianoneill/go-bootstrap/pkg/domain/config"
	domainhttp "github.com/damianoneill/go-bootstrap/pkg/domain/http"
	domainlog "github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	domaintracing "github.com/damianoneill/go-bootstrap/pkg/domain/tracing"
)

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	MaxHeaderSize   int
	ShutdownTimeout time.Duration
	TLSEnabled      bool
	TLSCertFile     string
	TLSKeyFile      string
}

// ServerHooks provides hooks for testing server lifecycle
type ServerHooks struct {
	ListenAndServe func() error                // Optional hook for testing server startup
	Shutdown       func(context.Context) error // Optional hook for testing server shutdown
}

// Service represents a bootstrapped application with core capabilities.
type Service struct {
	logger    domainlog.Logger
	config    domainconfig.Store
	router    domainhttp.Router
	tracer    domaintracing.Provider
	startTime time.Time
	server    *http.Server
	deps      Dependencies
	hooks     *ServerHooks // Optional test hooks
	opts      Options
}

// NewService creates a new bootstrap service with all domain capabilities
func NewService(opts Options, deps Dependencies, hooks *ServerHooks) (*Service, error) {

	// Validate and set option defaults
	if err := validateOptions(&opts); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	svc := &Service{
		deps:      deps,
		startTime: time.Now(),
		hooks:     hooks,
		opts:      opts,
	}

	if err := svc.initConfig(opts); err != nil {
		return nil, err
	}

	if err := svc.initLogger(opts); err != nil {
		return nil, err
	}

	if err := svc.initTracing(opts); err != nil {
		return nil, err
	}

	if err := svc.initRouter(opts); err != nil {
		return nil, err
	}

	return svc, nil
}

// LoadServerConfig loads server configuration from the config store
func (s *Service) LoadServerConfig() (ServerConfig, error) {
	var cfg ServerConfig
	var ok bool

	// Load required port configuration
	cfg.Port, ok = s.config.GetInt("server.http.port")
	if !ok {
		return cfg, fmt.Errorf("server port not configured")
	}

	// Load timeouts with defaults
	cfg.ReadTimeout, ok = s.config.GetDuration("server.http.read_timeout")
	if !ok {
		cfg.ReadTimeout = 15 * time.Second
	}

	cfg.WriteTimeout, ok = s.config.GetDuration("server.http.write_timeout")
	if !ok {
		cfg.WriteTimeout = 15 * time.Second
	}

	cfg.IdleTimeout, ok = s.config.GetDuration("server.http.idle_timeout")
	if !ok {
		cfg.IdleTimeout = 60 * time.Second
	}

	cfg.ShutdownTimeout, ok = s.config.GetDuration("server.http.shutdown_timeout")
	if !ok {
		cfg.ShutdownTimeout = 15 * time.Second
	}

	// Load size limits
	cfg.MaxHeaderSize, ok = s.config.GetInt("server.http.max_header_size")
	if !ok {
		cfg.MaxHeaderSize = 1 << 20 // 1MB default
	}

	// Load TLS configuration
	cfg.TLSEnabled, _ = s.config.GetBool("server.tls.enabled")
	if cfg.TLSEnabled {
		cfg.TLSCertFile, _ = s.config.GetString("server.tls.cert_file")
		cfg.TLSKeyFile, _ = s.config.GetString("server.tls.key_file")
	}

	return cfg, nil
}

// createServer creates a new HTTP server with the given configuration
func (s *Service) createServer(cfg ServerConfig) (*http.Server, error) {
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Port),
		Handler:        s.router,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		IdleTimeout:    cfg.IdleTimeout,
		MaxHeaderBytes: cfg.MaxHeaderSize,
	}

	if err := s.configureTLS(server, cfg); err != nil {
		return nil, fmt.Errorf("configuring TLS: %w", err)
	}

	if s.opts.Server.PreStart != nil {
		if err := s.opts.Server.PreStart(server); err != nil {
			return nil, fmt.Errorf("pre-start hook: %w", err)
		}
	}

	return server, nil
}

// Start initializes and starts the HTTP server
// Start initializes and starts the HTTP server
func (s *Service) Start() error {
	cfg, err := s.LoadServerConfig()
	if err != nil {
		return fmt.Errorf("loading server config: %w", err)
	}

	server, err := s.createServer(cfg)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}
	s.server = server

	s.logger.InfoWith("Starting server", domainlog.Fields{
		"address":     s.server.Addr,
		"tls_enabled": cfg.TLSEnabled,
		"tls_cert":    cfg.TLSCertFile,
		"tls_key":     cfg.TLSKeyFile,
	})

	// Check if we should use TLS
	if cfg.TLSEnabled && cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		s.logger.InfoWith("Starting TLS server", domainlog.Fields{
			"cert_file": cfg.TLSCertFile,
			"key_file":  cfg.TLSKeyFile,
		})

		// Use hooks if provided, otherwise use standard ListenAndServeTLS
		if s.hooks != nil && s.hooks.ListenAndServe != nil {
			if err := s.hooks.ListenAndServe(); err != http.ErrServerClosed {
				return fmt.Errorf("server error: %w", err)
			}
		} else {
			if err := s.server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != http.ErrServerClosed {
				return fmt.Errorf("server error: %w", err)
			}
		}
	} else {
		// Use test hook if provided, otherwise use standard ListenAndServe
		if s.hooks != nil && s.hooks.ListenAndServe != nil {
			if err := s.hooks.ListenAndServe(); err != http.ErrServerClosed {
				return fmt.Errorf("server error: %w", err)
			}
		} else {
			if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
				return fmt.Errorf("server error: %w", err)
			}
		}
	}

	return nil
}

// Shutdown gracefully stops the service
func (s *Service) Shutdown(ctx context.Context) error {
	s.logger.Info("Starting graceful shutdown")

	// Get shutdown timeout from config
	cfg, err := s.LoadServerConfig()
	if err != nil {
		return fmt.Errorf("loading shutdown config: %w", err)
	}

	// Create timeout context using configured shutdown timeout
	ctx, cancel := context.WithTimeout(ctx, cfg.ShutdownTimeout)
	defer cancel()

	// Use test hook if provided, otherwise use standard Shutdown
	shutdown := s.server.Shutdown
	if s.hooks != nil && s.hooks.Shutdown != nil {
		shutdown = s.hooks.Shutdown
	}

	if err := shutdown(ctx); err != nil {
		s.logger.ErrorWith("Shutdown error", domainlog.Fields{
			"error": err.Error(),
		})
		return fmt.Errorf("server shutdown: %w", err)
	}

	if s.tracer != nil {
		if err := s.tracer.Shutdown(ctx); err != nil {
			s.logger.ErrorWith("Tracer shutdown error", domainlog.Fields{
				"error": err.Error(),
			})
			return fmt.Errorf("tracer shutdown: %w", err)
		}
	}

	s.logger.Info("Server stopped")
	return nil
}

// Router returns the service's router
func (s *Service) Router() domainhttp.Router {
	return s.router
}

// Config returns the service's configuration store
func (s *Service) Config() domainconfig.Store {
	return s.config
}

// Logger returns the service's logger
func (s *Service) Logger() domainlog.Logger {
	return s.logger
}

// validateOptions ensures all required options are set and defaults are applied
func validateOptions(opts *Options) error {
	if opts.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}

	// Set defaults for service identity
	if opts.Version == "" {
		opts.Version = "dev"
	}
	if opts.EnvPrefix == "" {
		opts.EnvPrefix = opts.ServiceName
	}

	// Set defaults for logging
	if opts.LogLevel == "" {
		opts.LogLevel = domainlog.InfoLevel
	}

	// Set defaults for server
	if opts.Server.ShutdownTimeout == 0 {
		opts.Server.ShutdownTimeout = 15 * time.Second
	}
	if opts.Server.ReadTimeout == 0 {
		opts.Server.ReadTimeout = 15 * time.Second
	}
	if opts.Server.WriteTimeout == 0 {
		opts.Server.WriteTimeout = 15 * time.Second
	}
	if opts.Server.IdleTimeout == 0 {
		opts.Server.IdleTimeout = 60 * time.Second
	}
	if opts.Server.MaxHeaderSize == 0 {
		opts.Server.MaxHeaderSize = 1 << 20 // 1MB default
	}
	if opts.Server.Port == 0 {
		opts.Server.Port = 8080
	}

	// Set defaults for tracing
	if opts.TracingSampleRate == 0 {
		opts.TracingSampleRate = 1.0
	}

	return nil
}

// createProbeHandlers creates probe handlers for Kubernetes health checks
func (s *Service) createProbeHandlers(opts Options) *domainhttp.ProbeHandlers {
	return &domainhttp.ProbeHandlers{
		LivenessCheck: func() domainhttp.ProbeResponse {
			return domainhttp.ProbeResponse{
				Status: "ok",
				Details: map[string]interface{}{
					"version": opts.Version,
					"uptime":  time.Since(s.startTime).String(),
				},
			}
		},
		ReadinessCheck: func() domainhttp.ProbeResponse {
			return domainhttp.ProbeResponse{
				Status: "ok",
				Details: map[string]interface{}{
					"startup_time": s.startTime.Format(time.RFC3339),
				},
			}
		},
		StartupCheck: func() domainhttp.ProbeResponse {
			return domainhttp.ProbeResponse{
				Status: "ok",
			}
		},
	}
}

func (s *Service) configureTLS(server *http.Server, cfg ServerConfig) error {
	if !cfg.TLSEnabled {
		return nil
	}

	// Load TLS certificate if certificate files are provided
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("loading TLS cert/key: %w", err)
		}

		if server.TLSConfig == nil {
			server.TLSConfig = &tls.Config{}
		}
		server.TLSConfig.Certificates = []tls.Certificate{cert}
	}

	// Ensure minimum TLS version
	if server.TLSConfig == nil {
		server.TLSConfig = &tls.Config{}
	}
	if server.TLSConfig.MinVersion == 0 {
		server.TLSConfig.MinVersion = tls.VersionTLS12
	}

	return nil
}
