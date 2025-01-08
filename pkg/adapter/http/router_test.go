// pkg/adapter/http/router_test.go
package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	domainhttp "github.com/damianoneill/go-bootstrap/pkg/domain/http"
	mocklog "github.com/damianoneill/go-bootstrap/pkg/domain/logging/mocks"
	mockmetrics "github.com/damianoneill/go-bootstrap/pkg/domain/metrics/mocks"
	mocktracing "github.com/damianoneill/go-bootstrap/pkg/domain/tracing/mocks"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	assert.NotNil(t, factory)
}

func TestNewRouter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name      string
		options   []domainhttp.Option
		setupMock func(*mockmetrics.MockFactory)
		wantErr   bool
	}{
		{
			name: "success with minimal options",
			options: []domainhttp.Option{
				domainhttp.WithService("test-service", "1.0"),
			},
			setupMock: func(mf *mockmetrics.MockFactory) {},
			wantErr:   false,
		},
		{
			name:      "error without service name",
			options:   []domainhttp.Option{},
			setupMock: func(mf *mockmetrics.MockFactory) {},
			wantErr:   true,
		},
		{
			name: "success with metrics",
			options: []domainhttp.Option{
				domainhttp.WithService("test-service", "1.0"),
			},
			setupMock: func(mf *mockmetrics.MockFactory) {
				collector := mockmetrics.NewMockCollector(ctrl)
				mf.EXPECT().NewCollector(gomock.Any()).Return(collector, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory()

			// Apply any metrics factory if test requires it
			if len(tt.options) > 0 && tt.name == "success with metrics" {
				metricsFactory := mockmetrics.NewMockFactory(ctrl)
				tt.setupMock(metricsFactory)
				tt.options = append(tt.options, domainhttp.WithMetricsFactory(metricsFactory))
			}

			router, err := factory.NewRouter(tt.options...)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, router)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, router)
			}
		})
	}
}

func TestRouterProbeEndpoints(t *testing.T) {
	factory := NewFactory()
	router, err := factory.NewRouter(
		domainhttp.WithService("test-service", "1.0"),
		domainhttp.WithProbeHandlers(domainhttp.DefaultProbeHandlers()),
	)
	assert.NoError(t, err)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   map[string]interface{}
	}{
		{
			name:       "liveness probe",
			path:       "/internal/health",
			wantStatus: http.StatusOK,
			wantBody: map[string]interface{}{
				"status": "ok",
			},
		},
		{
			name:       "readiness probe",
			path:       "/internal/ready",
			wantStatus: http.StatusOK,
			wantBody: map[string]interface{}{
				"status": "ok",
			},
		},
		{
			name:       "startup probe",
			path:       "/internal/startup",
			wantStatus: http.StatusOK,
			wantBody: map[string]interface{}{
				"status": "ok",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.path, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var got map[string]interface{}
			err := json.NewDecoder(w.Body).Decode(&got)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantBody["status"], got["status"])
		})
	}
}

func TestRouterMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := mocklog.NewMockLogger(ctrl)
	logger.EXPECT().InfoWith(gomock.Any(), gomock.Any()).AnyTimes()
	logger.EXPECT().WithContext(gomock.Any()).Return(logger).AnyTimes()

	collector := mockmetrics.NewMockCollector(ctrl)
	collector.EXPECT().CollectRequestMetrics(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).AnyTimes()

	metricsFactory := mockmetrics.NewMockFactory(ctrl)
	metricsFactory.EXPECT().NewCollector(gomock.Any()).Return(collector, nil)

	tracingProvider := mocktracing.NewMockProvider(ctrl)

	factory := NewFactory()
	router, err := factory.NewRouter(
		domainhttp.WithService("test-service", "1.0"),
		domainhttp.WithLogger(logger),
		domainhttp.WithMetricsFactory(metricsFactory),
		domainhttp.WithTracingProvider(tracingProvider),
	)
	assert.NoError(t, err)

	// Add test endpoint
	router.(*Router).Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test request
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRouterClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	collector := mockmetrics.NewMockCollector(ctrl)
	collector.EXPECT().Close().Return(nil)

	metricsFactory := mockmetrics.NewMockFactory(ctrl)
	metricsFactory.EXPECT().NewCollector(gomock.Any()).Return(collector, nil)

	factory := NewFactory()
	router, err := factory.NewRouter(
		domainhttp.WithService("test-service", "1.0"),
		domainhttp.WithMetricsFactory(metricsFactory),
	)
	assert.NoError(t, err)

	// Test close
	err = router.(*Router).Close(context.Background())
	assert.NoError(t, err)
}

func TestRouterObservabilityExclusions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := mocklog.NewMockLogger(ctrl)
	logger.EXPECT().InfoWith(gomock.Any(), gomock.Any()).AnyTimes()
	logger.EXPECT().WithContext(gomock.Any()).Return(logger).MinTimes(1)

	collector := mockmetrics.NewMockCollector(ctrl)
	collector.EXPECT().CollectRequestMetrics(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).MinTimes(1)

	metricsFactory := mockmetrics.NewMockFactory(ctrl)
	metricsFactory.EXPECT().NewCollector(gomock.Any()).Return(collector, nil)

	factory := NewFactory()
	router, err := factory.NewRouter(
		domainhttp.WithService("test-service", "1.0"),
		domainhttp.WithLogger(logger),
		domainhttp.WithMetricsFactory(metricsFactory),
		domainhttp.WithObservabilityExclusions(
			[]string{"/excluded"},
			[]string{"/excluded"},
		),
	)
	assert.NoError(t, err)

	// Add test endpoints
	router.(*Router).Get("/test", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Add small delay to ensure metrics are collected
		w.WriteHeader(http.StatusOK)
	})
	router.(*Router).Get("/excluded", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name string
		path string
	}{
		{
			name: "included path",
			path: "/test",
		},
		{
			name: "excluded path",
			path: "/excluded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.path, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
