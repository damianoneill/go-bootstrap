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
