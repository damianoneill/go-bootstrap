// pkg/usecase/bootstrap/types.go

package bootstrap

import (
	"time"

	domainconfig "github.com/damianoneill/go-bootstrap/pkg/domain/config"
	domainhttp "github.com/damianoneill/go-bootstrap/pkg/domain/http"
	"github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	domainlog "github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	domainmetrics "github.com/damianoneill/go-bootstrap/pkg/domain/metrics"
	domaintracing "github.com/damianoneill/go-bootstrap/pkg/domain/tracing"
)

// Dependencies contains all external dependencies required by the service.
type Dependencies struct {
	ConfigFactory  domainconfig.Factory
	LoggerFactory  domainlog.Factory
	RouterFactory  domainhttp.Factory
	TracerFactory  domaintracing.Factory
	MetricsFactory domainmetrics.Factory
}

// Options configures the bootstrap service.
type Options struct {
	// Service Identity
	ServiceName string
	Version     string

	// Configuration
	ConfigFile     string
	EnvPrefix      string
	ConfigDefaults map[string]interface{}

	// Logging
	LogLevel        logging.Level
	LogFields       logging.Fields
	EnableLogConfig bool // Whether to mount runtime log config endpoint

	// HTTP Server
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration

	// Router/Observability
	ExcludeFromLogging []string
	ExcludeFromTracing []string
	ProbeHandlers      *domainhttp.ProbeHandlers

	// Tracing
	TracingEndpoint    string
	TracingSampleRate  float64
	TracingPropagators []string
}
