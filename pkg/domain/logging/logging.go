// pkg/domain/logging/logging.go

// Package logging defines the core logging interfaces and options
// for structured logging support across the application.
package logging

import (
	"context"
	"net/http"

	"github.com/damianoneill/go-bootstrap/pkg/domain/options"
)

//go:generate mockgen -destination=mocks/mock_logger.go -package=mocks github.com/damianoneill/go-bootstrap/pkg/domain/logging Logger,LeveledLogger,RuntimeConfigurable,Factory

// Level represents logging severity levels.
type Level string

const (
	// DebugLevel logs debug or trace information
	DebugLevel Level = "debug"

	// InfoLevel logs general information about program execution
	InfoLevel Level = "info"

	// WarnLevel logs potentially harmful situations
	WarnLevel Level = "warn"

	// ErrorLevel logs error conditions
	ErrorLevel Level = "error"
)

// Fields represents structured logging key-value pairs.
// Keys should be strings, values can be any type.
type Fields map[string]interface{}

// LoggerOptions holds configuration for logger implementations.
type LoggerOptions struct {
	// Level sets the minimum logging level
	Level Level

	// ServiceName identifies the service in log output
	ServiceName string

	// Fields contains default fields added to all log entries
	Fields Fields
}

// Option is a function that modifies LoggerOptions
type Option = options.Option[LoggerOptions]

// DefaultOptions returns the default logger options
func DefaultOptions() LoggerOptions {
	return LoggerOptions{
		Level: InfoLevel,
	}
}

// WithDefaults ensures options have proper default values
func WithDefaults(opts *LoggerOptions) {
	if opts.Level == "" {
		opts.Level = InfoLevel
	}
}

// WithLevel sets the minimum logging level.
// Messages below this level will not be logged.
func WithLevel(level Level) Option {
	return options.OptionFunc[LoggerOptions](func(o *LoggerOptions) error {
		o.Level = level
		return nil
	})
}

// WithServiceName sets the service name that will be included
// in all log entries for identification.
func WithServiceName(name string) Option {
	return options.OptionFunc[LoggerOptions](func(o *LoggerOptions) error {
		o.ServiceName = name
		return nil
	})
}

// WithFields sets default fields that will be included
// in all log entries from this logger.
func WithFields(fields Fields) Option {
	return options.OptionFunc[LoggerOptions](func(o *LoggerOptions) error {
		o.Fields = fields
		return nil
	})
}

// Logger defines the core logging interface.
// It provides both simple logging methods and methods that accept
// additional structured fields.
type Logger interface {
	// Simple logging methods without additional fields

	// Debug logs a message at debug level
	Debug(msg string)

	// Info logs a message at info level
	Info(msg string)

	// Warn logs a message at warn level
	Warn(msg string)

	// Error logs a message at error level
	Error(msg string)

	// Structured logging methods with additional fields

	// DebugWith logs a message at debug level with additional fields
	DebugWith(msg string, fields Fields)

	// InfoWith logs a message at info level with additional fields
	InfoWith(msg string, fields Fields)

	// WarnWith logs a message at warn level with additional fields
	WarnWith(msg string, fields Fields)

	// ErrorWith logs a message at error level with additional fields
	ErrorWith(msg string, fields Fields)

	// Context methods for creating derived loggers

	// With returns a new Logger with additional default fields
	With(fields Fields) Logger

	// WithContext returns a new Logger with context information
	// This typically adds trace IDs and other context metadata
	WithContext(ctx context.Context) Logger
}

// LeveledLogger extends Logger with level management capabilities.
type LeveledLogger interface {
	Logger

	// SetLevel changes the minimum logging level
	SetLevel(level Level)

	// GetLevel returns the current minimum logging level
	GetLevel() Level
}

// RuntimeConfigurable represents a logger that supports runtime
// configuration through an HTTP endpoint.
type RuntimeConfigurable interface {
	// GetConfigHandler returns an http.Handler for runtime configuration
	// This typically allows changing log levels via HTTP requests
	GetConfigHandler() http.Handler
}

// Factory creates new logger instances
type Factory interface {
	// NewLogger creates a new LeveledLogger with the given options
	NewLogger(opts ...Option) (LeveledLogger, error)
}
