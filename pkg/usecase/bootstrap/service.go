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
	domainmetrics "github.com/damianoneill/go-bootstrap/pkg/domain/metrics"
	domaintracing "github.com/damianoneill/go-bootstrap/pkg/domain/tracing"
)

// Dependencies contains all external dependencies required by the service.
// This makes it easy to inject real or mock implementations for testing.
type Dependencies struct {
	ConfigFactory  domainconfig.Factory
	LoggerFactory  domainlog.Factory
	RouterFactory  domainhttp.Factory
	TracerFactory  domaintracing.Factory
	MetricsFactory domainmetrics.Factory
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
}

// Options configures the bootstrap service.
type Options struct {
	ServiceName     string
	Version         string
	ConfigFile      string
	EnvPrefix       string
	LogLevel        domainlog.Level
	TracingEndpoint string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// NewService creates a new bootstrap service.
func NewService(opts Options, deps Dependencies) (*Service, error) {
	if err := validateOptions(&opts); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	svc := &Service{
		deps:      deps,
		startTime: time.Now(),
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

func (s *Service) initConfig(opts Options) error {
	cfgOpts := []domainconfig.Option{
		domainconfig.WithEnvPrefix(opts.EnvPrefix),
		domainconfig.WithDefaults(map[string]interface{}{
			"server.http.port":          opts.Port,
			"server.http.read_timeout":  opts.ReadTimeout,
			"server.http.write_timeout": opts.WriteTimeout,
			"logging.level":             string(opts.LogLevel),
		}),
	}
	if opts.ConfigFile != "" {
		cfgOpts = append(cfgOpts, domainconfig.WithConfigFile(opts.ConfigFile))
	}

	store, err := s.deps.ConfigFactory.NewStore(cfgOpts...)
	if err != nil {
		return fmt.Errorf("creating config store: %w", err)
	}
	s.config = store
	return nil
}

func (s *Service) initLogger(opts Options) error {
	logger, err := s.deps.LoggerFactory.NewLogger(
		domainlog.WithLevel(opts.LogLevel),
		domainlog.WithServiceName(opts.ServiceName),
		domainlog.WithFields(domainlog.Fields{
			"version": opts.Version,
		}),
	)
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}
	s.logger = logger
	return nil
}

func (s *Service) initTracing(opts Options) error {
	if opts.TracingEndpoint == "" {
		return nil
	}

	provider, err := s.deps.TracerFactory.NewProvider(
		domaintracing.WithServiceName(opts.ServiceName),
		domaintracing.WithServiceVersion(opts.Version),
		domaintracing.WithCollectorEndpoint(opts.TracingEndpoint),
		domaintracing.WithExporterType(domaintracing.GRPCExporter),
		domaintracing.WithInsecure(true),
		domaintracing.WithSamplingRate(1.0),
		domaintracing.WithDefaultPropagators(),
	)
	if err != nil {
		return fmt.Errorf("creating tracer: %w", err)
	}
	s.tracer = provider
	return nil
}

func (s *Service) initRouter(opts Options) error {
	probeHandlers := s.createProbeHandlers(opts)
	routerOpts := []domainhttp.Option{
		domainhttp.WithService(opts.ServiceName, opts.Version),
		domainhttp.WithLogger(s.logger),
		domainhttp.WithProbeHandlers(probeHandlers),
		domainhttp.WithObservabilityExclusions(
			[]string{"/internal/*", "/metrics"},
			[]string{"/internal/*", "/metrics"},
		),
	}

	// Add metrics factory if configured
	if s.deps.MetricsFactory != nil {
		routerOpts = append(routerOpts, domainhttp.WithMetricsFactory(s.deps.MetricsFactory))
	}

	if s.tracer != nil {
		routerOpts = append(routerOpts, domainhttp.WithTracingProvider(s.tracer))
	}

	router, err := s.deps.RouterFactory.NewRouter(routerOpts...)
	if err != nil {
		return fmt.Errorf("creating router: %w", err)
	}
	s.router = router

	// Add logger config endpoint if supported
	if configurable, ok := s.logger.(domainlog.RuntimeConfigurable); ok {
		router.Mount("/internal/logging", configurable.GetConfigHandler())
		s.logger.InfoWith("Registered logger config endpoint", domainlog.Fields{
			"path": "/internal/logging",
		})
	}

	return nil
}

// Start initializes and starts the HTTP server
func (s *Service) Start() error {
	// Get port from config
	port, ok := s.config.GetInt("server.http.port")
	if !ok {
		return fmt.Errorf("server port not configured")
	}

	// Get timeouts from config
	readTimeout, ok := s.config.GetDuration("server.http.read_timeout")
	if !ok {
		readTimeout = 15 * time.Second // fallback to default
	}

	writeTimeout, ok := s.config.GetDuration("server.http.write_timeout")
	if !ok {
		writeTimeout = 15 * time.Second // fallback to default
	}

	addr := fmt.Sprintf(":%d", port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	s.logger.InfoWith("Starting server", domainlog.Fields{
		"address": addr,
	})

	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the service
func (s *Service) Shutdown(ctx context.Context) error {
	s.logger.Info("Starting graceful shutdown")

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.ErrorWith("Shutdown error", domainlog.Fields{
			"error": err.Error(),
		})
		return err
	}

	if s.tracer != nil {
		if err := s.tracer.Shutdown(ctx); err != nil {
			s.logger.ErrorWith("Tracer shutdown error", domainlog.Fields{
				"error": err.Error(),
			})
			return err
		}
	}

	s.logger.Info("Server stopped")
	return nil
}

// Accessors for components
func (s *Service) Router() domainhttp.Router  { return s.router }
func (s *Service) Config() domainconfig.Store { return s.config }
func (s *Service) Logger() domainlog.Logger   { return s.logger }

func validateOptions(opts *Options) error {
	if opts.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
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
	return nil
}

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
