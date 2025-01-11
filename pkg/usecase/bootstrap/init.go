// pkg/usecase/bootstrap/init.go

package bootstrap

import (
	"fmt"

	domainconfig "github.com/damianoneill/go-bootstrap/pkg/domain/config"
	domainhttp "github.com/damianoneill/go-bootstrap/pkg/domain/http"
	domainlog "github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	domaintracing "github.com/damianoneill/go-bootstrap/pkg/domain/tracing"
)

func (s *Service) initConfig(opts Options) error {
	cfgOpts := []domainconfig.Option{
		domainconfig.WithEnvPrefix(opts.EnvPrefix),
		domainconfig.WithDefaults(map[string]interface{}{
			"server.http.port":            opts.Server.Port,
			"server.http.read_timeout":    opts.Server.ReadTimeout,
			"server.http.write_timeout":   opts.Server.WriteTimeout,
			"server.http.idle_timeout":    opts.Server.IdleTimeout,
			"server.http.max_header_size": opts.Server.MaxHeaderSize,
			"server.tls.enabled":          opts.Server.TLSConfig != nil,
			"server.tls.cert_file":        opts.Server.TLSCertFile,
			"server.tls.key_file":         opts.Server.TLSKeyFile,
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
	// Build default fields
	fields := domainlog.Fields{
		"version": opts.Version,
	}

	// Merge user-provided fields if present
	if opts.LogFields != nil {
		for k, v := range opts.LogFields {
			fields[k] = v
		}
	}

	logger, err := s.deps.LoggerFactory.NewLogger(
		domainlog.WithLevel(opts.LogLevel),
		domainlog.WithServiceName(opts.ServiceName),
		domainlog.WithFields(fields),
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

	tracingOpts := []domaintracing.Option{
		domaintracing.WithServiceName(opts.ServiceName),
		domaintracing.WithServiceVersion(opts.Version),
		domaintracing.WithCollectorEndpoint(opts.TracingEndpoint),
		domaintracing.WithSamplingRate(opts.TracingSampleRate),
		domaintracing.WithInsecure(true),
	}

	if len(opts.TracingPropagators) > 0 {
		tracingOpts = append(tracingOpts,
			domaintracing.WithPropagatorTypes(opts.TracingPropagators))
	} else {
		tracingOpts = append(tracingOpts,
			domaintracing.WithDefaultPropagators())
	}

	provider, err := s.deps.TracerFactory.NewProvider(tracingOpts...)
	if err != nil {
		return fmt.Errorf("creating tracer: %w", err)
	}
	s.tracer = provider
	return nil
}

func (s *Service) initRouter(opts Options) error {
	probeHandlers := opts.ProbeHandlers
	if probeHandlers == nil {
		probeHandlers = s.createProbeHandlers(opts)
	}

	// Build up our router options slice
	routerOpts := []domainhttp.Option{
		domainhttp.WithService(opts.ServiceName, opts.Version),
		domainhttp.WithLogger(s.logger),
		domainhttp.WithProbeHandlers(probeHandlers),
	}

	// Default paths to exclude from observability if none specified
	excludeFromLogging := []string{"/internal/*", "/metrics"}
	excludeFromTracing := []string{"/internal/*", "/metrics"}

	// Override with user-specified paths if provided
	if len(opts.ExcludeFromLogging) > 0 {
		excludeFromLogging = opts.ExcludeFromLogging
	}
	if len(opts.ExcludeFromTracing) > 0 {
		excludeFromTracing = opts.ExcludeFromTracing
	}

	routerOpts = append(routerOpts,
		domainhttp.WithObservabilityExclusions(
			excludeFromLogging,
			excludeFromTracing,
		))

	// Add metrics factory if configured
	if s.deps.MetricsFactory != nil {
		routerOpts = append(routerOpts,
			domainhttp.WithMetricsFactory(s.deps.MetricsFactory))
	}

	if s.tracer != nil {
		routerOpts = append(routerOpts,
			domainhttp.WithTracingProvider(s.tracer))
	}

	// If user provided middleware ordering, add it
	if opts.Router.MiddlewareOrdering != nil {
		routerOpts = append(routerOpts,
			domainhttp.WithMiddlewareOrdering(opts.Router.MiddlewareOrdering))
	}

	router, err := s.deps.RouterFactory.NewRouter(routerOpts...)
	if err != nil {
		return fmt.Errorf("creating router: %w", err)
	}
	s.router = router

	// Add logger config endpoint if enabled
	if opts.EnableLogConfig {
		if configurable, ok := s.logger.(domainlog.RuntimeConfigurable); ok {
			router.Mount("/internal/logging", configurable.GetConfigHandler())
			s.logger.InfoWith("Registered logger config endpoint",
				domainlog.Fields{"path": "/internal/logging"})
		}
	}

	// Add config endpoint if enabled and store supports masking
	if opts.EnableConfigViewer {
		if maskedStore, ok := s.config.(domainconfig.MaskedStore); ok {
			// Create config viewing endpoint with default mask strategy
			strategy := &domainconfig.DefaultMaskStrategy{
				SensitiveKeys: []string{"password", "secret", "key", "token", "credential"},
				MaskPattern:   "******",
			}
			router.Mount("/internal/config", maskedStore.GetConfigHandler(strategy))
			s.logger.InfoWith("Registered config viewing endpoint",
				domainlog.Fields{"path": "/internal/config"})
		}
	}

	return nil
}
