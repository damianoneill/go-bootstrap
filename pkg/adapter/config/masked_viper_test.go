// pkg/adapter/config/masked_viper_test.go

package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainconfig "github.com/damianoneill/go-bootstrap/pkg/domain/config"
)

func TestViperStore_GetMaskedConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		strategy domainconfig.MaskStrategy
		want     map[string]interface{}
	}{
		{
			name: "masks sensitive values",
			config: map[string]interface{}{
				"database": map[string]interface{}{
					"host":     "localhost",
					"password": "secret123",
				},
				"api_key": "abcd1234",
			},
			strategy: &domainconfig.DefaultMaskStrategy{
				SensitiveKeys: []string{"password", "key"},
				MaskPattern:   "******",
			},
			want: map[string]interface{}{
				"database": map[string]interface{}{
					"host":     "localhost",
					"password": "******",
				},
				"api_key": "******",
			},
		},
		{
			name: "default strategy when nil provided",
			config: map[string]interface{}{
				"database": map[string]interface{}{
					"password": "secret123",
				},
			},
			strategy: nil,
			want: map[string]interface{}{
				"database": map[string]interface{}{
					"password": "******",
				},
			},
		},
		{
			name: "preserves nested structures",
			config: map[string]interface{}{
				"server": map[string]interface{}{
					"port": 8080,
					"tls": map[string]interface{}{
						"enabled":  true,
						"key_file": "/path/to/key.pem",
					},
				},
			},
			strategy: &domainconfig.DefaultMaskStrategy{
				SensitiveKeys: []string{"key"},
				MaskPattern:   "******",
			},
			want: map[string]interface{}{
				"server": map[string]interface{}{
					"port": 8080,
					"tls": map[string]interface{}{
						"enabled":  true,
						"key_file": "******",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create store with test config
			f := NewFactory()
			store, err := f.NewStore()
			require.NoError(t, err)

			// Set test values
			for k, v := range tt.config {
				err := store.Set(k, v)
				require.NoError(t, err)
			}

			// Get masked config
			got, err := store.GetMaskedConfig(tt.strategy)
			require.NoError(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestViperStore_ConfigHandler(t *testing.T) {
	// Create store with test config
	f := NewFactory()
	store, err := f.NewStore()
	require.NoError(t, err)

	testConfig := map[string]interface{}{
		"database": map[string]interface{}{
			"host":     "localhost",
			"password": "secret123",
		},
	}

	// Set test values
	for k, v := range testConfig {
		err := store.Set(k, v)
		require.NoError(t, err)
	}

	strategy := &domainconfig.DefaultMaskStrategy{
		SensitiveKeys: []string{"password"},
		MaskPattern:   "******",
	}

	handler := store.GetConfigHandler(strategy)

	t.Run("GET returns masked config", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/internal/config", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var got map[string]interface{}
		err = json.NewDecoder(rec.Body).Decode(&got)
		require.NoError(t, err)

		want := map[string]interface{}{
			"database": map[string]interface{}{
				"host":     "localhost",
				"password": "******",
			},
		}

		assert.Equal(t, want, got)
	})

	t.Run("POST returns method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/internal/config", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})
}
