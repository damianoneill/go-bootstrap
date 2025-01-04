package logging

import (
	"context"
	"net/http"

	"github.com/damianoneill/go-bootstrap/pkg/domain/options"
)

// Level represents logging levels
type Level string

const (
	DebugLevel Level = "debug"
	InfoLevel  Level = "info"
	WarnLevel  Level = "warn"
	ErrorLevel Level = "error"
)

// Fields represents structured logging fields
type Fields map[string]interface{}

// LoggerOptions holds configuration for loggers
type LoggerOptions struct {
	Level       Level
	ServiceName string
	Fields      Fields
}

// Option is a logger option
type Option = options.Option[LoggerOptions]

// WithLevel sets the log level
func WithLevel(level Level) Option {
	return options.OptionFunc[LoggerOptions](func(o *LoggerOptions) error {
		o.Level = level
		return nil
	})
}

// WithServiceName sets the service name
func WithServiceName(name string) Option {
	return options.OptionFunc[LoggerOptions](func(o *LoggerOptions) error {
		o.ServiceName = name
		return nil
	})
}

// WithFields sets initial fields
func WithFields(fields Fields) Option {
	return options.OptionFunc[LoggerOptions](func(o *LoggerOptions) error {
		o.Fields = fields
		return nil
	})
}

// Logger defines the core logging interface
type Logger interface {
	Info(msg string, fields Fields)
	Error(msg string, fields Fields)
	Debug(msg string, fields Fields)
	Warn(msg string, fields Fields)
	With(fields Fields) Logger
	WithContext(ctx context.Context) Logger
}

// LeveledLogger extends Logger with level management
type LeveledLogger interface {
	Logger
	SetLevel(level Level)
	GetLevel() Level
}

// RuntimeConfigurable represents a logger that supports runtime configuration
type RuntimeConfigurable interface {
	GetConfigHandler() http.Handler
}

// LoggerFactory creates new logger instances
type LoggerFactory interface {
	NewLogger(opts ...Option) (LeveledLogger, error)
}
