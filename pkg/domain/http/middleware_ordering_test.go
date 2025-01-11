// pkg/domain/http/middleware_ordering_test.go
package http

import (
	"testing"
)

func TestMiddlewareOrderingValidation(t *testing.T) {
	tests := []struct {
		name    string
		order   []MiddlewareCategory
		wantErr bool
	}{
		{
			name: "valid order with all required categories",
			order: []MiddlewareCategory{
				CoreMiddleware,
				SecurityMiddleware,
				ApplicationMiddleware,
				ObservabilityMiddleware,
			},
			wantErr: false,
		},
		{
			name: "missing required category",
			order: []MiddlewareCategory{
				CoreMiddleware,
				ApplicationMiddleware,
			},
			wantErr: true,
		},
		{
			name: "duplicate category",
			order: []MiddlewareCategory{
				CoreMiddleware,
				SecurityMiddleware,
				CoreMiddleware,
				ObservabilityMiddleware,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMiddlewareOrdering(tt.order)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMiddlewareOrdering() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
