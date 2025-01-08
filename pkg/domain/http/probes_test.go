// pkg/domain/http/probe_test.go
package http_test

import (
	"encoding/json"
	"testing"

	"github.com/damianoneill/go-bootstrap/pkg/domain/http"
)

func TestNewProbeResponse(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		details map[string]interface{}
		want    http.ProbeResponse
	}{
		{
			name:   "response with status only",
			status: "ok",
			want: http.ProbeResponse{
				Status:  "ok",
				Details: nil,
			},
		},
		{
			name:   "response with status and details",
			status: "ok",
			details: map[string]interface{}{
				"version": "1.0.0",
				"uptime":  "1h",
			},
			want: http.ProbeResponse{
				Status: "ok",
				Details: map[string]interface{}{
					"version": "1.0.0",
					"uptime":  "1h",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := http.NewProbeResponse(tt.status, tt.details)

			if got.Status != tt.want.Status {
				t.Errorf("Status = %v, want %v", got.Status, tt.want.Status)
			}

			if tt.details == nil {
				if got.Details != nil {
					t.Error("Details = non-nil, want nil")
				}
				return
			}

			gotJSON, err := json.Marshal(got.Details)
			if err != nil {
				t.Fatalf("Failed to marshal got details: %v", err)
			}

			wantJSON, err := json.Marshal(tt.want.Details)
			if err != nil {
				t.Fatalf("Failed to marshal want details: %v", err)
			}

			if string(gotJSON) != string(wantJSON) {
				t.Errorf("Details = %v, want %v", string(gotJSON), string(wantJSON))
			}
		})
	}
}

func TestDefaultProbeHandlers(t *testing.T) {
	handlers := http.DefaultProbeHandlers()

	tests := []struct {
		name       string
		check      func() http.ProbeResponse
		wantStatus string
	}{
		{
			name:       "liveness check",
			check:      handlers.LivenessCheck,
			wantStatus: "ok",
		},
		{
			name:       "readiness check",
			check:      handlers.ReadinessCheck,
			wantStatus: "ok",
		},
		{
			name:       "startup check",
			check:      handlers.StartupCheck,
			wantStatus: "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.check()

			if got.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", got.Status, tt.wantStatus)
			}

			if got.Details != nil {
				t.Error("Details = non-nil, want nil for default handlers")
			}
		})
	}
}

func TestCustomProbeHandlers(t *testing.T) {
	customDetails := map[string]interface{}{
		"version": "1.0.0",
		"uptime":  "1h",
	}

	handlers := &http.ProbeHandlers{
		LivenessCheck: func() http.ProbeResponse {
			return http.NewProbeResponse("ok", customDetails)
		},
		ReadinessCheck: func() http.ProbeResponse {
			return http.NewProbeResponse("not_ready", nil)
		},
		StartupCheck: func() http.ProbeResponse {
			return http.NewProbeResponse("starting", customDetails)
		},
	}

	tests := []struct {
		name        string
		check       func() http.ProbeResponse
		wantStatus  string
		wantDetails bool
	}{
		{
			name:        "custom liveness check",
			check:       handlers.LivenessCheck,
			wantStatus:  "ok",
			wantDetails: true,
		},
		{
			name:        "custom readiness check",
			check:       handlers.ReadinessCheck,
			wantStatus:  "not_ready",
			wantDetails: false,
		},
		{
			name:        "custom startup check",
			check:       handlers.StartupCheck,
			wantStatus:  "starting",
			wantDetails: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.check()

			if got.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", got.Status, tt.wantStatus)
			}

			if tt.wantDetails {
				if got.Details == nil {
					t.Error("Details = nil, want non-nil")
				} else {
					for k, v := range customDetails {
						if got.Details[k] != v {
							t.Errorf("Details[%v] = %v, want %v", k, got.Details[k], v)
						}
					}
				}
			} else if got.Details != nil {
				t.Error("Details = non-nil, want nil")
			}
		})
	}
}
