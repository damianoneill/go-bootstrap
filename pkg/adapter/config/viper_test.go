// pkg/adapter/config/viper_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainconfig "github.com/damianoneill/go-bootstrap/pkg/domain/config"
)

func TestFactory_NewStore_WithFile(t *testing.T) {
	// Create temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	content := []byte(`
test_string: hello
test_int: 42
test_bool: true
test_duration: 1s
test_float: 3.14
test_slice:
  - one
  - two
`)

	err := os.WriteFile(configPath, content, 0644)
	require.NoError(t, err)

	// Create store with config file
	f := NewFactory()
	store, err := f.NewStore(domainconfig.WithConfigFile(configPath))
	require.NoError(t, err)

	// Test all value types
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "string values",
			testFunc: func(t *testing.T) {
				val, ok := store.GetString("test_string")
				assert.True(t, ok)
				assert.Equal(t, "hello", val)
			},
		},
		{
			name: "integer values",
			testFunc: func(t *testing.T) {
				val, ok := store.GetInt("test_int")
				assert.True(t, ok)
				assert.Equal(t, 42, val)
			},
		},
		{
			name: "boolean values",
			testFunc: func(t *testing.T) {
				val, ok := store.GetBool("test_bool")
				assert.True(t, ok)
				assert.True(t, val)
			},
		},
		{
			name: "duration values",
			testFunc: func(t *testing.T) {
				val, ok := store.GetDuration("test_duration")
				assert.True(t, ok)
				assert.Equal(t, time.Second, val)
			},
		},
		{
			name: "float values",
			testFunc: func(t *testing.T) {
				val, ok := store.GetFloat64("test_float")
				assert.True(t, ok)
				assert.Equal(t, 3.14, val)
			},
		},
		{
			name: "slice values",
			testFunc: func(t *testing.T) {
				val, ok := store.GetStringSlice("test_slice")
				assert.True(t, ok)
				assert.Equal(t, []string{"one", "two"}, val)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func TestFactory_NewStore_WithEnv(t *testing.T) {
	// Set test environment variables
	t.Setenv("TEST_CONFIG_VALUE", "from_env")

	// Create store with env prefix
	f := NewFactory()
	store, err := f.NewStore(domainconfig.WithEnvPrefix("TEST_CONFIG"))
	require.NoError(t, err)

	// Verify env var is readable
	val, ok := store.GetString("value")
	assert.True(t, ok)
	assert.Equal(t, "from_env", val)
}

func TestFactory_NewStore_WithDefaults(t *testing.T) {
	defaults := map[string]interface{}{
		"default_key":  "default_value",
		"override_key": "default",
	}

	// Create store with defaults
	f := NewFactory()
	store, err := f.NewStore(domainconfig.WithDefaults(defaults))
	require.NoError(t, err)

	// Test default value
	val, ok := store.GetString("default_key")
	assert.True(t, ok)
	assert.Equal(t, "default_value", val)

	// Override default and verify
	err = store.Set("override_key", "new_value")
	require.NoError(t, err)

	val, ok = store.GetString("override_key")
	assert.True(t, ok)
	assert.Equal(t, "new_value", val)
}

func TestStore_UnmarshalKey(t *testing.T) {
	config := `
app:
  name: test-app
  port: 8080
  features:
    enabled: true
`
	// Create temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	// Create store with config file
	f := NewFactory()
	store, err := f.NewStore(domainconfig.WithConfigFile(configPath))
	require.NoError(t, err)

	// Define struct matching config
	type AppConfig struct {
		Name     string
		Port     int
		Features struct {
			Enabled bool
		}
	}

	// Unmarshal and verify
	var appConfig AppConfig
	err = store.UnmarshalKey("app", &appConfig)
	require.NoError(t, err)

	assert.Equal(t, "test-app", appConfig.Name)
	assert.Equal(t, 8080, appConfig.Port)
	assert.True(t, appConfig.Features.Enabled)
}
