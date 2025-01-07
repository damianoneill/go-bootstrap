// pkg/domain/logging/logging_test.go
package logging

import (
	"testing"
)

func defaultOptions() LoggerOptions {
	return LoggerOptions{
		Level: InfoLevel,
	}
}

func TestLoggerOptions(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		expected LoggerOptions
		wantErr  bool
	}{
		{
			name:    "default options",
			options: []Option{},
			expected: LoggerOptions{
				Level: InfoLevel,
			},
		},
		{
			name: "set level",
			options: []Option{
				WithLevel(DebugLevel),
			},
			expected: LoggerOptions{
				Level: DebugLevel,
			},
		},
		{
			name: "set service name",
			options: []Option{
				WithServiceName("test-service"),
			},
			expected: LoggerOptions{
				Level:       InfoLevel,
				ServiceName: "test-service",
			},
		},
		{
			name: "set fields",
			options: []Option{
				WithFields(Fields{"key": "value"}),
			},
			expected: LoggerOptions{
				Level:  InfoLevel,
				Fields: Fields{"key": "value"},
			},
		},
		{
			name: "set multiple options",
			options: []Option{
				WithLevel(ErrorLevel),
				WithServiceName("test-service"),
				WithFields(Fields{"key": "value"}),
			},
			expected: LoggerOptions{
				Level:       ErrorLevel,
				ServiceName: "test-service",
				Fields:      Fields{"key": "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := defaultOptions()
			for _, opt := range tt.options {
				err := opt.ApplyOption(&opts)
				if (err != nil) != tt.wantErr {
					t.Errorf("ApplyOption() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}

			if opts.Level != tt.expected.Level {
				t.Errorf("Level = %v, want %v", opts.Level, tt.expected.Level)
			}
			if opts.ServiceName != tt.expected.ServiceName {
				t.Errorf("ServiceName = %v, want %v", opts.ServiceName, tt.expected.ServiceName)
			}
			if len(opts.Fields) != len(tt.expected.Fields) {
				t.Errorf("Fields length = %v, want %v", len(opts.Fields), len(tt.expected.Fields))
			}
			for k, v := range tt.expected.Fields {
				if opts.Fields[k] != v {
					t.Errorf("Fields[%s] = %v, want %v", k, opts.Fields[k], v)
				}
			}
		})
	}
}
