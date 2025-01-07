// pkg/domain/options/options_test.go
package options

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testConfig is a sample configuration struct used for testing
type testConfig struct {
	Name    string
	Port    int
	Enabled bool
}

// createOption creates a test option with error handling
func createOption(name string, port int, enabled bool, shouldError bool) Option[testConfig] {
	return OptionFunc[testConfig](func(c *testConfig) error {
		if shouldError {
			return errors.New("option error")
		}
		c.Name = name
		c.Port = port
		c.Enabled = enabled
		return nil
	})
}

func TestApply(t *testing.T) {
	tests := []struct {
		name      string
		opts      []Option[testConfig]
		expected  testConfig
		wantError bool
	}{
		{
			name: "no options",
			opts: nil,
			expected: testConfig{
				Name:    "",
				Port:    0,
				Enabled: false,
			},
			wantError: false,
		},
		{
			name: "single option",
			opts: []Option[testConfig]{
				createOption("test", 8080, true, false),
			},
			expected: testConfig{
				Name:    "test",
				Port:    8080,
				Enabled: true,
			},
			wantError: false,
		},
		{
			name: "multiple options",
			opts: []Option[testConfig]{
				createOption("test1", 8080, true, false),
				createOption("test2", 9090, false, false),
			},
			expected: testConfig{
				Name:    "test2",
				Port:    9090,
				Enabled: false,
			},
			wantError: false,
		},
		{
			name: "error in option",
			opts: []Option[testConfig]{
				createOption("test", 8080, true, true),
			},
			expected:  testConfig{},
			wantError: true,
		},
		{
			name: "error stops further options",
			opts: []Option[testConfig]{
				createOption("test1", 8080, true, false),
				createOption("test2", 9090, false, true),
				createOption("test3", 7070, true, false),
			},
			expected: testConfig{
				Name:    "test1",
				Port:    8080,
				Enabled: true,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &testConfig{}
			err := Apply(cfg, tt.opts...)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, *cfg)
			}
		})
	}
}

func TestOptionFunc_ApplyOption(t *testing.T) {
	t.Run("successful apply", func(t *testing.T) {
		opt := OptionFunc[testConfig](func(c *testConfig) error {
			c.Name = "test"
			return nil
		})

		cfg := &testConfig{}
		err := opt.ApplyOption(cfg)

		assert.NoError(t, err)
		assert.Equal(t, "test", cfg.Name)
	})

	t.Run("error handling", func(t *testing.T) {
		expectedErr := errors.New("test error")
		opt := OptionFunc[testConfig](func(c *testConfig) error {
			return expectedErr
		})

		cfg := &testConfig{}
		err := opt.ApplyOption(cfg)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("nil option function", func(t *testing.T) {
		var opt OptionFunc[testConfig]
		cfg := &testConfig{
			Name: "original",
			Port: 8080,
		}

		err := opt.ApplyOption(cfg)
		assert.NoError(t, err)
		// Config should remain unchanged
		assert.Equal(t, "original", cfg.Name)
		assert.Equal(t, 8080, cfg.Port)
	})
}

func TestApply_NilOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option[testConfig]
		expected testConfig
	}{
		{
			name:     "nil option slice",
			opts:     nil,
			expected: testConfig{},
		},
		{
			name: "nil options in slice",
			opts: []Option[testConfig]{
				nil,
				createOption("test", 8080, true, false),
				nil,
			},
			expected: testConfig{
				Name:    "test",
				Port:    8080,
				Enabled: true,
			},
		},
		{
			name: "nil function options",
			opts: []Option[testConfig]{
				OptionFunc[testConfig](nil),
				createOption("test", 8080, true, false),
				OptionFunc[testConfig](nil),
			},
			expected: testConfig{
				Name:    "test",
				Port:    8080,
				Enabled: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &testConfig{}
			err := Apply(cfg, tt.opts...)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, *cfg)
		})
	}
}

// Example of using options in real-world scenario
type serverConfig struct {
	Host    string
	Port    int
	TLS     bool
	Timeout int
}

func withHost(host string) Option[serverConfig] {
	return OptionFunc[serverConfig](func(c *serverConfig) error {
		if host == "" {
			return errors.New("host cannot be empty")
		}
		c.Host = host
		return nil
	})
}

func withPort(port int) Option[serverConfig] {
	return OptionFunc[serverConfig](func(c *serverConfig) error {
		if port < 1 || port > 65535 {
			return errors.New("port must be between 1 and 65535")
		}
		c.Port = port
		return nil
	})
}

func TestServerConfig(t *testing.T) {
	tests := []struct {
		name      string
		opts      []Option[serverConfig]
		expected  serverConfig
		wantError bool
	}{
		{
			name: "valid configuration",
			opts: []Option[serverConfig]{
				withHost("localhost"),
				withPort(8080),
			},
			expected: serverConfig{
				Host: "localhost",
				Port: 8080,
			},
			wantError: false,
		},
		{
			name: "empty host",
			opts: []Option[serverConfig]{
				withHost(""),
				withPort(8080),
			},
			expected:  serverConfig{},
			wantError: true,
		},
		{
			name: "invalid port",
			opts: []Option[serverConfig]{
				withHost("localhost"),
				withPort(70000),
			},
			expected:  serverConfig{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &serverConfig{}
			err := Apply(cfg, tt.opts...)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Host, cfg.Host)
				assert.Equal(t, tt.expected.Port, cfg.Port)
			}
		})
	}
}
