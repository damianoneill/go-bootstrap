// pkg/usecase/bootstrap/service.go

package bootstrap

import (
	"context"
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
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
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

	cfg.Port, ok = s.config.GetInt("server.http.port")
	if !ok {
		return cfg, fmt.Errorf("server port not configured")
	}

	cfg.ReadTimeout, ok = s.config.GetDuration("server.http.read_timeout")
	if !ok {
		cfg.ReadTimeout = 15 * time.Second
	}

	cfg.WriteTimeout, ok = s.config.GetDuration("server.http.write_timeout")
	if !ok {
		cfg.WriteTimeout = 15 * time.Second
	}

	return cfg, nil
}

// createServer creates a new HTTP server with the given configuration
func (s *Service) createServer(cfg ServerConfig) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
}

// Start initializes and starts the HTTP server
func (s *Service) Start() error {
	cfg, err := s.LoadServerConfig()
	if err != nil {
		return fmt.Errorf("loading server config: %w", err)
	}

	s.server = s.createServer(cfg)

	s.logger.InfoWith("Starting server", domainlog.Fields{
		"address": s.server.Addr,
	})

	// Use test hook if provided, otherwise use standard ListenAndServe
	listenAndServe := s.server.ListenAndServe
	if s.hooks != nil && s.hooks.ListenAndServe != nil {
		listenAndServe = s.hooks.ListenAndServe
	}

	if err := listenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Shutdown gracefully stops the service
func (s *Service) Shutdown(ctx context.Context) error {
	s.logger.Info("Starting graceful shutdown")

	// Create timeout context using configured shutdown timeout
	ctx, cancel := context.WithTimeout(ctx, s.opts.ShutdownTimeout)
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

	// Set defaults
	if opts.Version == "" {
		opts.Version = "dev"
	}
	if opts.EnvPrefix == "" {
		opts.EnvPrefix = opts.ServiceName
	}
	if opts.LogLevel == "" {
		opts.LogLevel = domainlog.InfoLevel
	}
	if opts.ShutdownTimeout == 0 {
		opts.ShutdownTimeout = 15 * time.Second
	}
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = 15 * time.Second
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = 15 * time.Second
	}
	if opts.Port == 0 {
		opts.Port = 8080
	}
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
