// pkg/usecase/bootstrap/service_test.go

package bootstrap_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	configmocks "github.com/damianoneill/go-bootstrap/pkg/domain/config/mocks"
	domainhttp "github.com/damianoneill/go-bootstrap/pkg/domain/http"
	httpmocks "github.com/damianoneill/go-bootstrap/pkg/domain/http/mocks"
	domainlog "github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	logmocks "github.com/damianoneill/go-bootstrap/pkg/domain/logging/mocks"
	metricsmocks "github.com/damianoneill/go-bootstrap/pkg/domain/metrics/mocks"
	"github.com/damianoneill/go-bootstrap/pkg/domain/tracing"
	tracingmocks "github.com/damianoneill/go-bootstrap/pkg/domain/tracing/mocks"
	"github.com/damianoneill/go-bootstrap/pkg/usecase/bootstrap"
)

type testDeps struct {
	configFactory  *configmocks.MockFactory
	configStore    *configmocks.MockMaskedStore
	loggerFactory  *logmocks.MockFactory
	logger         *logmocks.MockLeveledLogger
	routerFactory  *httpmocks.MockFactory
	router         *httpmocks.MockRouter
	tracerFactory  *tracingmocks.MockFactory
	tracer         *tracingmocks.MockProvider
	metricsFactory *metricsmocks.MockFactory
	ctrl           *gomock.Controller
}

func newTestDeps(t *testing.T) *testDeps {
	ctrl := gomock.NewController(t)

	d := &testDeps{
		ctrl:           ctrl,
		configFactory:  configmocks.NewMockFactory(ctrl),
		configStore:    configmocks.NewMockMaskedStore(ctrl),
		loggerFactory:  logmocks.NewMockFactory(ctrl),
		logger:         logmocks.NewMockLeveledLogger(ctrl),
		routerFactory:  httpmocks.NewMockFactory(ctrl),
		router:         httpmocks.NewMockRouter(ctrl),
		tracerFactory:  tracingmocks.NewMockFactory(ctrl),
		tracer:         tracingmocks.NewMockProvider(ctrl),
		metricsFactory: metricsmocks.NewMockFactory(ctrl),
	}

	return d
}

func (d *testDeps) setupBasicMockExpectations(allowPort bool) {
	// Update mock expectations to use MaskedStore
	d.configFactory.EXPECT().
		NewStore(gomock.Any()).
		Return(d.configStore, nil).
		AnyTimes()

	// Common config expectations - skip port if not allowed
	if allowPort {
		d.configStore.EXPECT().GetInt("server.http.port").Return(8080, true).AnyTimes()
	}
	d.configStore.EXPECT().GetDuration("server.http.read_timeout").Return(15*time.Second, true).AnyTimes()
	d.configStore.EXPECT().GetDuration("server.http.write_timeout").Return(15*time.Second, true).AnyTimes()
	d.configStore.EXPECT().GetDuration("server.http.idle_timeout").Return(60*time.Second, true).AnyTimes()
	d.configStore.EXPECT().GetDuration("server.http.shutdown_timeout").Return(15*time.Second, true).AnyTimes()
	d.configStore.EXPECT().GetInt("server.http.max_header_size").Return(1<<20, true).AnyTimes()
	d.configStore.EXPECT().GetBool("server.tls.enabled").Return(false, true).AnyTimes()

	// Add expectations for config viewing if enabled
	d.configStore.EXPECT().
		GetConfigHandler(gomock.Any()).
		Return(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).
		AnyTimes()
	d.configStore.EXPECT().
		GetMaskedConfig(gomock.Any()).
		Return(make(map[string]interface{}), nil).
		AnyTimes()
}

func (d *testDeps) setupLoggerExpectations() {
	d.loggerFactory.EXPECT().NewLogger(gomock.Any()).
		DoAndReturn(func(opts ...domainlog.Option) (domainlog.LeveledLogger, error) {
			testOpts := &domainlog.LoggerOptions{}
			for _, opt := range opts {
				if err := opt.ApplyOption(testOpts); err != nil {
					return nil, err
				}
			}
			return d.logger, nil
		}).AnyTimes()
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name    string
		opts    bootstrap.Options
		setup   func(*testDeps)
		wantErr bool
	}{
		{
			name: "successful initialization with minimal options",
			opts: bootstrap.Options{
				ServiceName: "test-service",
				Version:     "1.0.0",
			},
			setup: func(d *testDeps) {
				d.setupBasicMockExpectations(true)
				d.setupLoggerExpectations()
				d.routerFactory.EXPECT().NewRouter(gomock.Any()).Return(d.router, nil)
			},
		},
		{
			name: "error when service name is empty",
			opts: bootstrap.Options{
				Version: "1.0.0",
			},
			setup:   func(d *testDeps) {},
			wantErr: true,
		},
		{
			name: "error creating config store",
			opts: bootstrap.Options{
				ServiceName: "test-service",
				Version:     "1.0.0",
			},
			setup: func(d *testDeps) {
				d.configFactory.EXPECT().NewStore(gomock.Any()).Return(nil, errors.New("config error"))
			},
			wantErr: true,
		},
		{
			name: "error creating logger",
			opts: bootstrap.Options{
				ServiceName: "test-service",
				Version:     "1.0.0",
			},
			setup: func(d *testDeps) {
				d.configFactory.EXPECT().NewStore(gomock.Any()).Return(d.configStore, nil)
				d.loggerFactory.EXPECT().NewLogger(gomock.Any()).Return(nil, errors.New("logger error"))
			},
			wantErr: true,
		},
		{
			name: "successful initialization with tracing",
			opts: bootstrap.Options{
				ServiceName:     "test-service",
				Version:         "1.0.0",
				TracingEndpoint: "localhost:4317",
			},
			setup: func(d *testDeps) {
				d.setupBasicMockExpectations(true)
				d.setupLoggerExpectations()
				d.routerFactory.EXPECT().NewRouter(gomock.Any()).Return(d.router, nil)
				d.tracerFactory.EXPECT().NewProvider(gomock.Any()).Return(d.tracer, nil)
			},
		},
		{
			name: "successful initialization with custom log fields",
			opts: bootstrap.Options{
				ServiceName: "test-service",
				Version:     "1.0.0",
				LogFields: domainlog.Fields{
					"environment": "test",
					"region":      "us-west",
				},
			},
			setup: func(d *testDeps) {
				d.setupBasicMockExpectations(true)
				d.loggerFactory.EXPECT().NewLogger(gomock.Any()).
					DoAndReturn(func(opts ...domainlog.Option) (domainlog.LeveledLogger, error) {
						testOpts := &domainlog.LoggerOptions{}
						for _, opt := range opts {
							err := opt.ApplyOption(testOpts)
							require.NoError(t, err)
						}

						// Verify fields include both version and custom fields
						expectedFields := domainlog.Fields{
							"version":     "1.0.0",
							"environment": "test",
							"region":      "us-west",
						}
						assert.Equal(t, expectedFields, testOpts.Fields)

						return d.logger, nil
					}).Times(1)
				d.routerFactory.EXPECT().NewRouter(gomock.Any()).Return(d.router, nil)
			},
		},
		{
			name: "initialization with custom observability exclusions",
			opts: bootstrap.Options{
				ServiceName:        "test-service",
				Version:            "1.0.0",
				ExcludeFromLogging: []string{"/health/*", "/custom/*"},
				ExcludeFromTracing: []string{"/health/*", "/metrics"},
			},
			setup: func(d *testDeps) {
				d.setupBasicMockExpectations(true)
				d.setupLoggerExpectations()

				d.routerFactory.EXPECT().NewRouter(gomock.Any()).
					DoAndReturn(func(opts ...domainhttp.Option) (domainhttp.Router, error) {
						testOpts := &domainhttp.RouterOptions{}
						for _, opt := range opts {
							err := opt.ApplyOption(testOpts)
							require.NoError(t, err)
						}
						assert.Equal(t, []string{"/health/*", "/custom/*"}, testOpts.ExcludeFromLogging)
						assert.Equal(t, []string{"/health/*", "/metrics"}, testOpts.ExcludeFromTracing)
						return d.router, nil
					})
			},
		},
		{
			name: "initialization with full tracing configuration",
			opts: bootstrap.Options{
				ServiceName:        "test-service",
				Version:            "1.0.0",
				TracingEndpoint:    "localhost:4317",
				TracingSampleRate:  0.5,
				TracingPropagators: []string{"tracecontext", "baggage"},
			},
			setup: func(d *testDeps) {
				d.setupBasicMockExpectations(true)
				d.setupLoggerExpectations()
				d.routerFactory.EXPECT().NewRouter(gomock.Any()).Return(d.router, nil)

				d.tracerFactory.EXPECT().NewProvider(gomock.Any()).
					DoAndReturn(func(opts ...tracing.Option) (tracing.Provider, error) {
						testOpts := &tracing.Options{}
						for _, opt := range opts {
							err := opt.ApplyOption(testOpts)
							require.NoError(t, err)
						}
						assert.Equal(t, "test-service", testOpts.ServiceName)
						assert.Equal(t, "1.0.0", testOpts.ServiceVersion)
						assert.Equal(t, "localhost:4317", testOpts.CollectorEndpoint)
						assert.Equal(t, 0.5, testOpts.SamplingRate)
						assert.True(t, testOpts.Insecure)
						return d.tracer, nil
					})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := newTestDeps(t)
			if tt.setup != nil {
				tt.setup(deps)
			}

			svc, err := bootstrap.NewService(tt.opts, bootstrap.Dependencies{
				ConfigFactory:  deps.configFactory,
				LoggerFactory:  deps.loggerFactory,
				RouterFactory:  deps.routerFactory,
				TracerFactory:  deps.tracerFactory,
				MetricsFactory: deps.metricsFactory,
			}, nil) // No hooks needed for initialization tests

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, svc)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, svc)

			// Verify service components are initialized
			assert.NotNil(t, svc.Logger())
			assert.NotNil(t, svc.Config())
			assert.NotNil(t, svc.Router())
		})
	}
}

func TestService_Lifecycle(t *testing.T) {
	tests := []struct {
		name    string
		opts    bootstrap.Options
		setup   func(*testDeps)
		wantErr bool
		runTest func(*testing.T, *bootstrap.Service, chan error)
	}{
		{
			name: "successful start and shutdown",
			opts: bootstrap.Options{
				ServiceName: "test-service",
				Version:     "1.0.0",
			},
			setup: func(d *testDeps) {
				d.setupBasicMockExpectations(true)
				d.setupLoggerExpectations()
				d.routerFactory.EXPECT().NewRouter(gomock.Any()).Return(d.router, nil)

				d.logger.EXPECT().InfoWith(gomock.Any(), gomock.Any()).Times(1)
				d.logger.EXPECT().Info("Starting graceful shutdown").Times(1)
				d.logger.EXPECT().Info("Server stopped").Times(1)
			},
			runTest: func(t *testing.T, svc *bootstrap.Service, startErrCh chan error) {
				time.Sleep(100 * time.Millisecond)
				err := svc.Shutdown(context.Background())
				assert.NoError(t, err)
				select {
				case err := <-startErrCh:
					assert.NoError(t, err)
				default:
				}
			},
		},
		{
			name: "error getting port configuration",
			opts: bootstrap.Options{
				ServiceName: "test-service",
				Version:     "1.0.0",
			},
			setup: func(d *testDeps) {
				d.setupBasicMockExpectations(false)
				d.setupLoggerExpectations()
				d.routerFactory.EXPECT().NewRouter(gomock.Any()).Return(d.router, nil)

				// Use gomock.Eq for exact string matching
				d.configStore.EXPECT().
					GetInt(gomock.Eq("server.http.port")).
					Return(0, false)
			},
			wantErr: true,
			runTest: func(t *testing.T, svc *bootstrap.Service, startErrCh chan error) {
				// Wait for error
				select {
				case err := <-startErrCh:
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "server port not configured")
				case <-time.After(time.Second):
					t.Fatal("timeout waiting for error")
				}
			},
		},
		{
			name: "shutdown respects configured timeout",
			opts: bootstrap.Options{
				ServiceName: "test-service",
				Version:     "1.0.0",
				Server: bootstrap.ServerOptions{
					ShutdownTimeout: 5 * time.Second,
				},
			},
			setup: func(d *testDeps) {
				d.setupBasicMockExpectations(true)
				d.setupLoggerExpectations()
				d.routerFactory.EXPECT().NewRouter(gomock.Any()).Return(d.router, nil)

				// Logger expectations
				d.logger.EXPECT().InfoWith(gomock.Any(), gomock.Any()).Times(1)
				d.logger.EXPECT().Info("Starting graceful shutdown").Times(1)
				d.logger.EXPECT().Info("Server stopped").Times(1)
			},
			runTest: func(t *testing.T, svc *bootstrap.Service, startErrCh chan error) {
				// Wait briefly to let server "start"
				time.Sleep(100 * time.Millisecond)

				// Create parent context with longer timeout
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				start := time.Now()
				err := svc.Shutdown(ctx)
				duration := time.Since(start)

				assert.NoError(t, err)
				// Verify shutdown completed within configured timeout
				assert.True(t, duration < 5*time.Second)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := newTestDeps(t)
			if tt.setup != nil {
				tt.setup(deps)
			}

			hooks := &bootstrap.ServerHooks{
				ListenAndServe: func() error {
					// simulate immediate server stop
					return http.ErrServerClosed
				},
				Shutdown: func(context.Context) error {
					return nil
				},
			}

			svc, err := bootstrap.NewService(tt.opts, bootstrap.Dependencies{
				ConfigFactory:  deps.configFactory,
				LoggerFactory:  deps.loggerFactory,
				RouterFactory:  deps.routerFactory,
				TracerFactory:  deps.tracerFactory,
				MetricsFactory: deps.metricsFactory,
			}, hooks)

			require.NoError(t, err)
			require.NotNil(t, svc)

			// Test startup
			startErrCh := make(chan error, 1)
			go func() {
				startErrCh <- svc.Start()
			}()

			tt.runTest(t, svc, startErrCh)
		})
	}
}
