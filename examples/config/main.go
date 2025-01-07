package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/damianoneill/go-bootstrap/pkg/adapter/config"
	domainconfig "github.com/damianoneill/go-bootstrap/pkg/domain/config"
)

// Configuration structs matching YAML structure
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Tracing  TracingConfig  `mapstructure:"tracing"`
	Features FeaturesConfig `mapstructure:"features"`
}

type ServerConfig struct {
	HTTP struct {
		Host            string        `mapstructure:"host"`
		Port            int           `mapstructure:"port"`
		ReadTimeout     time.Duration `mapstructure:"read_timeout"`
		WriteTimeout    time.Duration `mapstructure:"write_timeout"`
		ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
		TLS             struct {
			Enabled  bool   `mapstructure:"enabled"`
			CertFile string `mapstructure:"cert_file"`
			KeyFile  string `mapstructure:"key_file"`
		} `mapstructure:"tls"`
	} `mapstructure:"http"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Name     string `mapstructure:"name"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Pool     struct {
		MaxOpen     int           `mapstructure:"max_open"`
		MaxIdle     int           `mapstructure:"max_idle"`
		MaxLifetime time.Duration `mapstructure:"max_lifetime"`
	} `mapstructure:"pool"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type MetricsConfig struct {
	Enabled            bool          `mapstructure:"enabled"`
	Path               string        `mapstructure:"path"`
	CollectionInterval time.Duration `mapstructure:"collection_interval"`
}

type TracingConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	Endpoint   string  `mapstructure:"endpoint"`
	SampleRate float64 `mapstructure:"sample_rate"`
}

type FeaturesConfig struct {
	Beta         []string `mapstructure:"beta"`
	Experimental []string `mapstructure:"experimental"`
}

func main() {
	fmt.Println("\n=== Example 1: Basic config loading ===")
	factory := config.NewFactory()
	store, err := factory.NewStore(
		domainconfig.WithConfigFile("examples/config/config.yaml"),
		domainconfig.WithEnvPrefix("APP"),
		domainconfig.WithDefaults(map[string]interface{}{
			"server.http.port": 8080,
			"logging.level":    "info",
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create config store: %v", err)
	}

	fmt.Println("\n=== Example 2: Reading individual values ===")
	if port, ok := store.GetInt("server.http.port"); ok {
		fmt.Printf("Server port: %d\n", port)
	}

	if level, ok := store.GetString("logging.level"); ok {
		fmt.Printf("Log level: %s\n", level)
	}

	if timeout, ok := store.GetDuration("server.http.read_timeout"); ok {
		fmt.Printf("Read timeout: %v\n", timeout)
	}

	fmt.Println("\n=== Example 3: Unmarshaling complete config ===")
	var cfg Config
	if err := store.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	fmt.Printf("Database pool settings: max_open=%d, max_idle=%d\n",
		cfg.Database.Pool.MaxOpen,
		cfg.Database.Pool.MaxIdle)

	fmt.Println("\n=== Example 4: Feature flags ===")
	if features, ok := store.GetStringSlice("features.beta"); ok {
		fmt.Printf("Beta features enabled: %v\n", features)
	}

	fmt.Println("\n=== Example 5: Environment variables override ===")
	os.Setenv("APP_SERVER_HTTP_PORT", "9090")
	if port, ok := store.GetInt("server.http.port"); ok {
		fmt.Printf("Server port from env: %d\n", port)
	}

	fmt.Println("\n=== Example 6: Validate configuration ===")
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	fmt.Println("Configuration validated successfully")

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	select {
	case <-sigChan:
		fmt.Println("Received shutdown signal")
	case <-time.After(10 * time.Second):
		fmt.Println("Example completed")
	}
}

func validateConfig(cfg Config) error {
	if cfg.Server.HTTP.Port < 1 || cfg.Server.HTTP.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.HTTP.Port)
	}

	if cfg.Server.HTTP.ReadTimeout < time.Second {
		return fmt.Errorf("read timeout too short: %v", cfg.Server.HTTP.ReadTimeout)
	}

	if cfg.Server.HTTP.TLS.Enabled {
		if cfg.Server.HTTP.TLS.CertFile == "" || cfg.Server.HTTP.TLS.KeyFile == "" {
			return fmt.Errorf("TLS enabled but cert/key files not specified")
		}
	}

	if cfg.Database.Pool.MaxIdle > cfg.Database.Pool.MaxOpen {
		return fmt.Errorf("max idle connections (%d) cannot exceed max open connections (%d)",
			cfg.Database.Pool.MaxIdle, cfg.Database.Pool.MaxOpen)
	}

	return nil
}
