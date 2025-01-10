// pkg/adapter/config/viper.go
package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"

	domainconfig "github.com/damianoneill/go-bootstrap/pkg/domain/config"
)

// Verify interface implementation
var _ domainconfig.MaskedStore = (*ViperStore)(nil)

func (s *ViperStore) GetConfigHandler(maskStrategy domainconfig.MaskStrategy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		config, err := s.GetMaskedConfig(maskStrategy)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func (s *ViperStore) GetMaskedConfig(maskStrategy domainconfig.MaskStrategy) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get all settings as a map
	allSettings := s.v.AllSettings()

	if maskStrategy == nil {
		// Use default strategy if none provided
		maskStrategy = &domainconfig.DefaultMaskStrategy{
			SensitiveKeys: []string{"password", "secret", "key", "token", "credential"},
			MaskPattern:   "******",
		}
	}

	// Recursively mask sensitive values
	masked := maskConfigMap("", allSettings, maskStrategy)
	return masked, nil
}

// Apply MaskStrategy to a config map recursively
func maskConfigMap(prefix string, config map[string]interface{}, strategy domainconfig.MaskStrategy) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range config {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		switch val := v.(type) {
		case map[string]interface{}:
			// Recurse into nested maps
			result[k] = maskConfigMap(fullKey, val, strategy)
		default:
			// Mask leaf values that match sensitive patterns
			result[k] = strategy.MaskValue(fullKey, v)
		}
	}

	return result
}

// ViperStore implements the Store interface using Viper
type ViperStore struct {
	v  *viper.Viper
	mu sync.RWMutex
}

// Factory creates Viper-backed stores
type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) NewStore(opts ...domainconfig.Option) (domainconfig.MaskedStore, error) {
	options := domainconfig.StoreOptions{}
	for _, opt := range opts {
		if err := opt.ApplyOption(&options); err != nil {
			return nil, fmt.Errorf("applying option: %w", err)
		}
	}

	v := viper.New()
	v.SetConfigType("yaml") // Default to YAML
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Apply options
	if options.ConfigFile != "" {
		v.SetConfigFile(options.ConfigFile)
	}
	if options.EnvPrefix != "" {
		v.SetEnvPrefix(options.EnvPrefix)
		v.AutomaticEnv()
	}
	if options.Defaults != nil {
		for key, value := range options.Defaults {
			v.SetDefault(key, value)
		}
	}

	store := &ViperStore{v: v}

	// Load config if file specified
	if options.ConfigFile != "" {
		if err := store.ReadConfig(); err != nil {
			return nil, err
		}
	}

	return store, nil
}

// ReadConfig loads the configuration file
func (s *ViperStore) ReadConfig() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.v.ReadInConfig(); err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	return nil
}

// Get methods implement Store interface
func (s *ViperStore) GetString(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.v.IsSet(key) {
		return "", false
	}
	return s.v.GetString(key), true
}

func (s *ViperStore) GetInt(key string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.v.IsSet(key) {
		return 0, false
	}
	return s.v.GetInt(key), true
}

func (s *ViperStore) GetBool(key string) (bool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.v.IsSet(key) {
		return false, false
	}
	return s.v.GetBool(key), true
}

func (s *ViperStore) GetDuration(key string) (time.Duration, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.v.IsSet(key) {
		return 0, false
	}
	return s.v.GetDuration(key), true
}

func (s *ViperStore) GetFloat64(key string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.v.IsSet(key) {
		return 0, false
	}
	return s.v.GetFloat64(key), true
}

func (s *ViperStore) GetStringSlice(key string) ([]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.v.IsSet(key) {
		return nil, false
	}
	return s.v.GetStringSlice(key), true
}

func (s *ViperStore) Set(key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.v.Set(key, value)
	return nil
}

func (s *ViperStore) IsSet(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.v.IsSet(key)
}

func (s *ViperStore) UnmarshalKey(key string, target interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.v.UnmarshalKey(key, target)
}

func (s *ViperStore) Unmarshal(target interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.v.Unmarshal(target)
}
