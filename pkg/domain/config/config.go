// pkg/domain/config/config.go

// Package config defines the core configuration interfaces and options
// for managing application configuration from various sources.
package config

import (
	"time"

	"github.com/damianoneill/go-bootstrap/pkg/domain/options"
)

//go:generate mockgen -destination=mocks/mock_config.go -package=mocks github.com/damianoneill/go-bootstrap/pkg/domain/config Store,MaskedStore,Factory

// Store defines the core configuration operations.
// It provides type-safe access to configuration values and supports
// dynamic configuration updates.
type Store interface {
	// GetString retrieves a string value by key.
	// Returns the value and true if found, empty string and false if not found.
	GetString(key string) (string, bool)

	// GetInt retrieves an integer value by key.
	// Returns the value and true if found, 0 and false if not found.
	GetInt(key string) (int, bool)

	// GetBool retrieves a boolean value by key.
	// Returns the value and true if found, false and false if not found.
	GetBool(key string) (bool, bool)

	// GetDuration retrieves a time.Duration value by key.
	// Returns the value and true if found, 0 and false if not found.
	GetDuration(key string) (time.Duration, bool)

	// GetFloat64 retrieves a float64 value by key.
	// Returns the value and true if found, 0.0 and false if not found.
	GetFloat64(key string) (float64, bool)

	// GetStringSlice retrieves a string slice value by key.
	// Returns the value and true if found, nil and false if not found.
	GetStringSlice(key string) ([]string, bool)

	// Set stores a value for the given key.
	// The value must be of a supported type.
	Set(key string, value interface{}) error

	// IsSet checks if a configuration key exists.
	IsSet(key string) bool

	// ReadConfig loads the configuration from the configured source.
	// This should be called after initial setup to load values.
	ReadConfig() error

	// UnmarshalKey decodes a specific config key into a struct.
	// The target must be a pointer to a struct.
	UnmarshalKey(key string, target interface{}) error

	// Unmarshal decodes the entire config into a struct.
	// The target must be a pointer to a struct.
	Unmarshal(target interface{}) error
}

// StoreOptions holds configuration for store implementations.
type StoreOptions struct {
	// ConfigFile is the path to the configuration file
	ConfigFile string

	// EnvPrefix is prepended to environment variables
	EnvPrefix string

	// Defaults holds default values for configuration keys
	Defaults map[string]interface{}
}

// Option is a function that modifies StoreOptions
type Option = options.Option[StoreOptions]

// WithConfigFile sets the path to the configuration file.
// The file format is determined by the file extension.
func WithConfigFile(path string) Option {
	return options.OptionFunc[StoreOptions](func(o *StoreOptions) error {
		o.ConfigFile = path
		return nil
	})
}

// WithEnvPrefix sets the prefix for environment variables.
// Environment variables will be checked by uppercasing the key
// and prepending this prefix.
func WithEnvPrefix(prefix string) Option {
	return options.OptionFunc[StoreOptions](func(o *StoreOptions) error {
		o.EnvPrefix = prefix
		return nil
	})
}

// WithDefaults sets the default configuration values.
// These values are used when a key is not found in other sources.
func WithDefaults(defaults map[string]interface{}) Option {
	return options.OptionFunc[StoreOptions](func(o *StoreOptions) error {
		o.Defaults = defaults
		return nil
	})
}

// Factory creates new store instances
type Factory interface {
	// NewStore creates a new configuration store with the given options.
	// The returned store will be ready to use but may need ReadConfig()
	// called to load values.
	NewStore(opts ...Option) (MaskedStore, error)
}
