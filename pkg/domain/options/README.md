# Options

Options are used to configure domain implementations. They are defined as structs in the domain package and are passed to the implementation constructors. Options are used to configure the implementation at runtime.

Implementation adapters can then extend the domain options with their own specific options. For example, a zap logging adapter could extend the domain logging options with zap specific options.

```go
// pkg/adapter/logging/zap.go
package logging

import domain "github.com/damianoneill/go-bootstrap/pkg/domain/logging"

// ZapLoggerOptions extends domain options with zap specifics
type ZapLoggerOptions struct {
    domain.LoggerOptions
    Development bool
}

// Option is a zap logger option
type Option = options.Option[ZapLoggerOptions]

// WithDevelopment sets development mode
func WithDevelopment(enabled bool) Option {
    return options.OptionFunc[ZapLoggerOptions](func(o *ZapLoggerOptions) error {
        o.Development = enabled
        return nil
    })
}
```

## Usage

Options are passed to the implementation constructor functions. The implementation constructor functions are responsible for applying the options to the implementation.

```go
// pkg/adapter/logging/zap.go
package logging

import domain "github.com/damianoneill/go-bootstrap/pkg/domain/logging"

// NewZapLogger creates a new zap logger

func NewZapLogger(opts ...Option) domain.Logger {
    options := ZapLoggerOptions{
        LoggerOptions: domain.LoggerOptions{
            Level: domain.InfoLevel,
        },
    }

    for _, opt := range opts {
        opt(&options)
    }

    // Create zap logger with options
    return zap.New(options.Development)
}
```

## Example

```go

import (
    "github.com/myorg/myapp/pkg/adapter/logging"
    domain "github.com/myorg/myapp/pkg/domain/logging"
)

func main() {
    logger := logging.NewZapLogger(logging.WithDevelopment(true))
    logger.Info("Hello, World!")
}
```
