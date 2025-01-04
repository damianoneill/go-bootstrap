package logging

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	domainlog "github.com/damianoneill/go-bootstrap/pkg/domain/logging"
	"github.com/damianoneill/go-bootstrap/pkg/domain/options"
)

type ZapLogger struct {
	logger *zap.Logger
	level  domainlog.Level
	atom   zap.AtomicLevel
}

type ZapOptions struct {
	domainlog.LoggerOptions
	Development bool
}

type ZapOption = options.Option[ZapOptions]

// WithDevelopment enables development mode
func WithDevelopment(enabled bool) ZapOption {
	return options.OptionFunc[ZapOptions](func(o *ZapOptions) error {
		o.Development = enabled
		return nil
	})
}

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) NewLogger(opts ...domainlog.Option) (domainlog.LeveledLogger, error) {
	// Initialize domain options
	zopts := ZapOptions{
		LoggerOptions: domainlog.LoggerOptions{
			Level: domainlog.InfoLevel,
		},
	}

	// Apply domain options
	for _, opt := range opts {
		if err := opt.ApplyOption(&zopts.LoggerOptions); err != nil {
			return nil, fmt.Errorf("applying domain options: %w", err)
		}
	}

	return f.createLogger(zopts)
}

// NewLoggerWithOptions creates a logger with both domain and Zap options
func (f *Factory) NewLoggerWithOptions(dopts []domainlog.Option, zopts []ZapOption) (domainlog.LeveledLogger, error) {
	options := ZapOptions{
		LoggerOptions: domainlog.LoggerOptions{
			Level: domainlog.InfoLevel,
		},
	}

	// Apply domain options
	for _, opt := range dopts {
		if err := opt.ApplyOption(&options.LoggerOptions); err != nil {
			return nil, fmt.Errorf("applying domain options: %w", err)
		}
	}

	// Apply zap-specific options
	for _, opt := range zopts {
		if err := opt.ApplyOption(&options); err != nil {
			return nil, fmt.Errorf("applying zap options: %w", err)
		}
	}

	return f.createLogger(options)
}

func (f *Factory) createLogger(zopts ZapOptions) (domainlog.LeveledLogger, error) {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(convertToZapLevel(zopts.Level)),
		Development:      zopts.Development,
		Sampling:         nil,
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		InitialFields:    make(map[string]interface{}),
	}

	if zopts.Development {
		config.Development = true
		config.DisableStacktrace = false
	} else {
		config.Development = false
		config.DisableStacktrace = true
	}

	logger, err := config.Build(
		zap.AddCallerSkip(1),
		zap.AddCaller(),
	)
	if err != nil {
		return nil, fmt.Errorf("building zap logger: %w", err)
	}

	if zopts.ServiceName != "" {
		logger = logger.With(zap.String("service", zopts.ServiceName))
	}

	if len(zopts.Fields) > 0 {
		logger = logger.With(convertFields(zopts.Fields)...)
	}

	return &ZapLogger{
		logger: logger,
		level:  zopts.Level,
		atom:   config.Level,
	}, nil
}

func (l *ZapLogger) Info(msg string, fields domainlog.Fields) {
	l.logger.Info(msg, convertFields(fields)...)
}

func (l *ZapLogger) Error(msg string, fields domainlog.Fields) {
	l.logger.Error(msg, convertFields(fields)...)
}

func (l *ZapLogger) Debug(msg string, fields domainlog.Fields) {
	l.logger.Debug(msg, convertFields(fields)...)
}

func (l *ZapLogger) Warn(msg string, fields domainlog.Fields) {
	l.logger.Warn(msg, convertFields(fields)...)
}

func (l *ZapLogger) With(fields domainlog.Fields) domainlog.Logger {
	return &ZapLogger{
		logger: l.logger.With(convertFields(fields)...),
		level:  l.level,
		atom:   l.atom,
	}
}

func (l *ZapLogger) WithContext(ctx context.Context) domainlog.Logger {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		spanCtx := span.SpanContext()
		if spanCtx.HasTraceID() {
			logger := l.logger.With(
				zap.String("trace_id", spanCtx.TraceID().String()),
				zap.String("span_id", spanCtx.SpanID().String()),
			)
			if spanCtx.IsSampled() {
				logger = logger.With(zap.Bool("sampled", true))
			}
			return &ZapLogger{
				logger: logger,
				level:  l.level,
				atom:   l.atom,
			}
		}
	}
	return l
}

func (l *ZapLogger) SetLevel(level domainlog.Level) {
	l.level = level
	l.atom.SetLevel(convertToZapLevel(level))
}

func (l *ZapLogger) GetLevel() domainlog.Level {
	return l.level
}

func (l *ZapLogger) GetConfigHandler() http.Handler {
	return l.atom
}

func convertToZapLevel(level domainlog.Level) zapcore.Level {
	switch level {
	case domainlog.DebugLevel:
		return zapcore.DebugLevel
	case domainlog.InfoLevel:
		return zapcore.InfoLevel
	case domainlog.WarnLevel:
		return zapcore.WarnLevel
	case domainlog.ErrorLevel:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func convertFields(fields domainlog.Fields) []zap.Field {
	if len(fields) == 0 {
		return nil
	}

	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return zapFields
}
