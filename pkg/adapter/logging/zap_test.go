// pkg/adapter/logging/zap_test.go
package logging

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	domainlog "github.com/damianoneill/go-bootstrap/pkg/domain/logging"
)

func newTestLogger(t *testing.T) (*ZapLogger, *observer.ObservedLogs) {
	core, obs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	return &ZapLogger{
		logger: logger,
		level:  domainlog.InfoLevel,
		atom:   zap.NewAtomicLevelAt(zap.InfoLevel),
	}, obs
}

func TestZapLogger_Levels(t *testing.T) {
	tests := []struct {
		name    string
		level   domainlog.Level
		logFunc func(l *ZapLogger, msg string)
		message string
		wantLog bool
	}{
		{
			name:    "debug not logged at info level",
			level:   domainlog.InfoLevel,
			logFunc: (*ZapLogger).Debug,
			message: "debug message",
			wantLog: false,
		},
		{
			name:    "info logged at info level",
			level:   domainlog.InfoLevel,
			logFunc: (*ZapLogger).Info,
			message: "info message",
			wantLog: true,
		},
		{
			name:    "warn logged at info level",
			level:   domainlog.InfoLevel,
			logFunc: (*ZapLogger).Warn,
			message: "warn message",
			wantLog: true,
		},
		{
			name:    "error logged at info level",
			level:   domainlog.InfoLevel,
			logFunc: (*ZapLogger).Error,
			message: "error message",
			wantLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, obs := newTestLogger(t)
			logger.SetLevel(tt.level)

			tt.logFunc(logger, tt.message)

			logs := obs.All()
			if tt.wantLog && len(logs) == 0 {
				t.Error("expected log entry but got none")
			}
			if !tt.wantLog && len(logs) > 0 {
				t.Error("expected no log entry but got one")
			}
			if tt.wantLog && len(logs) > 0 {
				assert.Equal(t, tt.message, logs[0].Message)
			}
		})
	}
}

func TestZapLogger_With(t *testing.T) {
	logger, obs := newTestLogger(t)

	fields := domainlog.Fields{
		"string": "value",
		"int":    123,
		"bool":   true,
	}

	derivedLogger := logger.With(fields)
	derivedLogger.Info("test message")

	logs := obs.All()
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, "test message", logs[0].Message)

	loggedFields := logs[0].ContextMap()
	assert.Equal(t, "value", loggedFields["string"])
	assert.Equal(t, int64(123), loggedFields["int"])
	assert.Equal(t, true, loggedFields["bool"])
}

func TestZapLogger_WithContext(t *testing.T) {
	logger, obs := newTestLogger(t)

	t.Run("with empty context", func(t *testing.T) {
		emptyCtx := context.Background()
		baseLogger := logger.WithContext(emptyCtx)
		baseLogger.Info("test message")

		// Check base context logs
		baseLogs := obs.All()
		if assert.Equal(t, 1, len(baseLogs), "should have one log message") {
			assert.Equal(t, "test message", baseLogs[0].Message)
		}

		// Clear the observer for next test
		obs.TakeAll()
	})

	t.Run("with trace context", func(t *testing.T) {
		// Set up tracer
		spanRecorder := tracetest.NewSpanRecorder()
		tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
		defer func() {
			if err := tracerProvider.Shutdown(context.Background()); err != nil {
				t.Errorf("Error shutting down tracer provider: %v", err)
			}
		}()

		tracer := tracerProvider.Tracer("test")

		// Create and end span
		ctx, span := tracer.Start(context.Background(), "test-span")
		tracedLogger := logger.WithContext(ctx)
		tracedLogger.Info("traced message")
		span.End()

		// Get traced logs
		tracedLogs := obs.All()
		if assert.Equal(t, 1, len(tracedLogs), "should have one traced log message") {
			logEntry := tracedLogs[0]
			assert.Equal(t, "traced message", logEntry.Message)

			// Verify trace fields are present
			loggedFields := logEntry.ContextMap()
			assert.NotEmpty(t, loggedFields["trace_id"], "trace_id should be present")
			assert.NotEmpty(t, loggedFields["span_id"], "span_id should be present")
		}

		// Verify spans were recorded - wait for span to be processed
		assert.Eventually(t, func() bool {
			spans := spanRecorder.Ended()
			return len(spans) > 0
		}, time.Second, 10*time.Millisecond, "span should be recorded")

		spans := spanRecorder.Ended()
		if assert.Equal(t, 1, len(spans), "should have one recorded span") {
			assert.Equal(t, "test-span", spans[0].Name())
		}
	})
}

func TestFactory_NewLogger(t *testing.T) {
	tests := []struct {
		name    string
		opts    []domainlog.Option
		zopts   []ZapOption
		wantErr bool
	}{
		{
			name:    "default options",
			opts:    nil,
			zopts:   nil,
			wantErr: false,
		},
		{
			name: "with service name",
			opts: []domainlog.Option{
				domainlog.WithServiceName("test-service"),
			},
			zopts:   nil,
			wantErr: false,
		},
		{
			name: "with development mode",
			opts: nil,
			zopts: []ZapOption{
				WithDevelopment(true),
			},
			wantErr: false,
		},
	}

	factory := NewFactory()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := factory.NewLoggerWithOptions(tt.opts, tt.zopts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLoggerWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.NotNil(t, logger)
				assert.Implements(t, (*domainlog.LeveledLogger)(nil), logger)
			}
		})
	}
}
