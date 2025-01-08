// pkg/usecase/bootstrap/types.go

package bootstrap

import (
	"time"

	domainconfig "github.com/damianoneill/go-bootstrap/pkg/domain/config"
	domainhttp "github.com/damianoneill/go-bootstrap/pkg/domain/http"
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
