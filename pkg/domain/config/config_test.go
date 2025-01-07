// pkg/domain/config/config_test.go
package config

import (
	"testing"
)

func TestWithConfigFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantPath string
	}{
		{
			name:     "sets config file path",
			path:     "/path/to/config.yaml",
			wantPath: "/path/to/config.yaml",
		},
		{
			name:     "empty path",
			path:     "",
			wantPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := WithConfigFile(tt.path)
			opts := StoreOptions{}

			if err := opt.ApplyOption(&opts); err != nil {
				t.Errorf("WithConfigFile() error = %v", err)
			}

			if opts.ConfigFile != tt.wantPath {
				t.Errorf("WithConfigFile() got = %v, want %v", opts.ConfigFile, tt.wantPath)
			}
		})
	}
}

func TestWithEnvPrefix(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		wantPrefix string
	}{
		{
			name:       "sets env prefix",
			prefix:     "APP",
			wantPrefix: "APP",
		},
		{
			name:       "empty prefix",
			prefix:     "",
			wantPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := WithEnvPrefix(tt.prefix)
			opts := StoreOptions{}

			if err := opt.ApplyOption(&opts); err != nil {
				t.Errorf("WithEnvPrefix() error = %v", err)
			}

			if opts.EnvPrefix != tt.wantPrefix {
				t.Errorf("WithEnvPrefix() got = %v, want %v", opts.EnvPrefix, tt.wantPrefix)
			}
		})
	}
}

func TestWithDefaults(t *testing.T) {
	tests := []struct {
		name         string
		defaults     map[string]interface{}
		wantDefaults map[string]interface{}
	}{
		{
			name: "sets defaults",
			defaults: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
			wantDefaults: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
		},
		{
			name:         "nil defaults",
			defaults:     nil,
			wantDefaults: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := WithDefaults(tt.defaults)
			opts := StoreOptions{}

			if err := opt.ApplyOption(&opts); err != nil {
				t.Errorf("WithDefaults() error = %v", err)
			}

			// Compare maps
			if len(opts.Defaults) != len(tt.wantDefaults) {
				t.Errorf("WithDefaults() got map len = %v, want %v",
					len(opts.Defaults), len(tt.wantDefaults))
			}

			for k, v := range tt.wantDefaults {
				if opts.Defaults[k] != v {
					t.Errorf("WithDefaults() got[%s] = %v, want %v", k, opts.Defaults[k], v)
				}
			}
		})
	}
}
