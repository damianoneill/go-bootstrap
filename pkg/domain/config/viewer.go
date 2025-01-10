// pkg/domain/config/viewer.go

package config

import (
	"net/http"
	"strings"
)

// MaskStrategy determines how sensitive data is masked
type MaskStrategy interface {
	// MaskValue masks potentially sensitive values
	// key is the full config path (e.g. "database.password")
	// value is the raw config value to potentially mask
	// Returns the masked value or original value if masking not needed
	MaskValue(key string, value interface{}) interface{}
}

// DefaultMaskStrategy provides standard masking for sensitive values
type DefaultMaskStrategy struct {
	// SensitiveKeys contains key patterns that should be masked (e.g. "password", "secret", "key")
	SensitiveKeys []string
	// MaskPattern is the string used to mask sensitive values (e.g. "******")
	MaskPattern string
}

// MaskValue implements MaskStrategy
func (s *DefaultMaskStrategy) MaskValue(key string, value interface{}) interface{} {
	if s.MaskPattern == "" {
		s.MaskPattern = "******"
	}
	for _, pattern := range s.SensitiveKeys {
		if containsInsensitive(key, pattern) {
			return s.MaskPattern
		}
	}
	return value
}

// containsInsensitive checks if str contains substr case-insensitively
func containsInsensitive(str, substr string) bool {
	str, substr = strings.ToLower(str), strings.ToLower(substr)
	return strings.Contains(str, substr)
}

// MaskedStore represents a config store that can expose masked config via HTTP
type MaskedStore interface {
	Store
	GetConfigHandler(maskStrategy MaskStrategy) http.Handler
	GetMaskedConfig(maskStrategy MaskStrategy) (map[string]interface{}, error)
}
