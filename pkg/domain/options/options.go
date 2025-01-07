// pkg/domain/options/options.go

// Package options provides generic option patterns for configurable types.
// It implements the functional options pattern in a type-safe way using generics.
package options

// Option represents a modification function for a configuration type T.
// It provides a type-safe way to configure structs using the functional options pattern.
type Option[T any] interface {
	// ApplyOption applies this option to the provided configuration object.
	// Returns an error if the option cannot be applied.
	ApplyOption(*T) error
}

// OptionFunc is a helper type that converts simple functions into Option instances.
// It provides a convenient way to create options without implementing the full interface.
type OptionFunc[T any] func(*T) error

// ApplyOption implements the Option interface for OptionFunc.
// It simply calls the function with the provided configuration object.
func (f OptionFunc[T]) ApplyOption(o *T) error {
	if f == nil {
		return nil // Treat nil function as a no-op
	}
	return f(o)
}

// Apply applies a sequence of options to a target configuration object.
// It processes options in order and returns on the first error encountered.
// This is the main entry point for applying options to a configuration object.
//
// Example usage:
//
//	type Config struct {
//	    Port int
//	    Host string
//	}
//
//	func WithPort(port int) Option[Config] {
//	    return OptionFunc[Config](func(c *Config) error {
//	        c.Port = port
//	        return nil
//	    })
//	}
//
//	config := &Config{}
//	err := Apply(config, WithPort(8080))
func Apply[T any](target *T, opts ...Option[T]) error {
	for _, opt := range opts {
		if opt == nil {
			continue // Skip nil options
		}
		if err := opt.ApplyOption(target); err != nil {
			return err
		}
	}
	return nil
}
