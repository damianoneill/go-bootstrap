// pkg/domain/config/config.go
package config

import (
	"context"
	"time"

	"github.com/damianoneill/go-bootstrap/pkg/domain/options"
)

// Store defines the core configuration operations
type Store interface {
	// Get methods return zero value and false if not found
	GetString(key string) (string, bool)
	GetInt(key string) (int, bool)
	GetBool(key string) (bool, bool)
	GetDuration(key string) (time.Duration, bool)
	GetFloat64(key string) (float64, bool)
	GetStringSlice(key string) ([]string, bool)

	// Set stores a value
	Set(key string, value interface{}) error

	// IsSet checks if a key exists
	IsSet(key string) bool

	// Load configuration
	ReadConfig() error

	// Unmarshal into structs
	UnmarshalKey(key string, target interface{}) error
	Unmarshal(target interface{}) error
}

// Watcher allows monitoring configuration changes
type Watcher interface {
	// Watch returns a channel that receives updates
	// The channel is closed when the context is canceled
	Watch(ctx context.Context, key string) (<-chan interface{}, error)
}

// StoreOptions holds configuration for stores
type StoreOptions struct {
	ConfigFile string
	EnvPrefix  string
	Defaults   map[string]interface{}
}

// Option is a store option
type Option = options.Option[StoreOptions]

// WithConfigFile sets the config file path
func WithConfigFile(path string) Option {
	return options.OptionFunc[StoreOptions](func(o *StoreOptions) error {
		o.ConfigFile = path
		return nil
	})
}

// WithEnvPrefix sets the environment variable prefix
func WithEnvPrefix(prefix string) Option {
	return options.OptionFunc[StoreOptions](func(o *StoreOptions) error {
		o.EnvPrefix = prefix
		return nil
	})
}

// WithDefaults sets default configuration values
func WithDefaults(defaults map[string]interface{}) Option {
	return options.OptionFunc[StoreOptions](func(o *StoreOptions) error {
		o.Defaults = defaults
		return nil
	})
}

// Factory creates new store instances
type Factory interface {
	NewStore(opts ...Option) (Store, error)
}
