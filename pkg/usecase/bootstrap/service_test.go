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
	httpmocks "github.com/damianoneill/go-bootstrap/pkg/domain/http/mocks"
	logmocks "github.com/damianoneill/go-bootstrap/pkg/domain/logging/mocks"
	metricsmocks "github.com/damianoneill/go-bootstrap/pkg/domain/metrics/mocks"
	tracingmocks "github.com/damianoneill/go-bootstrap/pkg/domain/tracing/mocks"
	"github.com/damianoneill/go-bootstrap/pkg/usecase/bootstrap"
)

type testDeps struct {
	configFactory  *configmocks.MockFactory
	configStore    *configmocks.MockStore
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
		configStore:    configmocks.NewMockStore(ctrl),
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
	// Basic expectations that most tests will need
	d.configFactory.EXPECT().NewStore(gomock.Any()).Return(d.configStore, nil).AnyTimes()
	d.loggerFactory.EXPECT().NewLogger(gomock.Any()).Return(d.logger, nil).AnyTimes()
	d.routerFactory.EXPECT().NewRouter(gomock.Any()).Return(d.router, nil).AnyTimes()

	// Common config expectations - skip port if not allowed
	if allowPort {
		d.configStore.EXPECT().GetInt("server.http.port").Return(8080, true).AnyTimes()
	}
	d.configStore.EXPECT().GetDuration("server.http.read_timeout").Return(15*time.Second, true).AnyTimes()
	d.configStore.EXPECT().GetDuration("server.http.write_timeout").Return(15*time.Second, true).AnyTimes()
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
				d.tracerFactory.EXPECT().NewProvider(gomock.Any()).Return(d.tracer, nil)
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
		setup   func(*testDeps)
		wantErr bool
		runTest func(*testing.T, *bootstrap.Service, chan error)
	}{
		{
			name: "successful start and shutdown",
			setup: func(d *testDeps) {
				d.setupBasicMockExpectations(true)
				d.logger.EXPECT().InfoWith(gomock.Any(), gomock.Any()).Times(1)
				d.logger.EXPECT().Info("Starting graceful shutdown").Times(1)
				d.logger.EXPECT().Info("Server stopped").Times(1)
			},
			runTest: func(t *testing.T, svc *bootstrap.Service, startErrCh chan error) {
				// Wait briefly to let server "start"
				time.Sleep(100 * time.Millisecond)

				// Test shutdown
				err := svc.Shutdown(context.Background())
				assert.NoError(t, err)

				// Verify no startup errors
				select {
				case err := <-startErrCh:
					assert.NoError(t, err)
				default:
				}
			},
		},
		{
			name: "error getting port configuration",
			setup: func(d *testDeps) {
				// Only setup basic expectations without port config
				d.setupBasicMockExpectations(false)

				// Expect port config to fail
				d.configStore.EXPECT().
					GetInt("server.http.port").
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

			svc, err := bootstrap.NewService(bootstrap.Options{
				ServiceName: "test-service",
				Version:     "1.0.0",
			}, bootstrap.Dependencies{
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
