// Package http provides domain interfaces for HTTP routing and service health probes.
package http

// ProbeResponse represents the result of a health check probe.
// It follows Kubernetes probe conventions while allowing additional
// details to be included in the response.
type ProbeResponse struct {
	// Status provides a string representation of the health state.
	// Common values include: "ok", "degraded", "failed".
	Status string `json:"status"`

	// Details contains additional probe information.
	// This can include timing data, dependency status, etc.
	Details map[string]interface{} `json:"details,omitempty"`
}

// ProbeCheck is a function that performs a health check and returns
// a ProbeResponse. It encapsulates the logic for determining the
// health state of a specific aspect of the service.
type ProbeCheck func() ProbeResponse

// ProbeHandlers contains the health check functions for Kubernetes probes.
// Each probe type serves a different purpose in determining service health
// and availability.
type ProbeHandlers struct {
	// LivenessCheck determines if the application is alive.
	// Kubernetes uses this to know when to restart a container.
	// A failed liveness check indicates the service is in an unrecoverable state.
	// This should check only the most basic service health.
	LivenessCheck ProbeCheck

	// ReadinessCheck determines if the application can serve traffic.
	// Kubernetes uses this to know when a container can receive requests.
	// A failed readiness check temporarily removes the service from the load balancer.
	// This should verify all required dependencies are available.
	ReadinessCheck ProbeCheck

	// StartupCheck determines if the application has completed startup.
	// Kubernetes uses this to know when a container has finished initialization.
	// A failed startup check prevents the service from receiving traffic
	// until initialization is complete.
	StartupCheck ProbeCheck
}

// DefaultProbeHandlers creates ProbeHandlers with sensible defaults.
// All probes return a healthy status with no additional details.
// This is suitable for basic services or initial development.
func DefaultProbeHandlers() *ProbeHandlers {
	defaultCheck := func() ProbeResponse {
		return ProbeResponse{
			Status: "ok",
		}
	}

	return &ProbeHandlers{
		LivenessCheck:  defaultCheck,
		ReadinessCheck: defaultCheck,
		StartupCheck:   defaultCheck,
	}
}

// NewProbeResponse creates a ProbeResponse with the given values.
// This is a convenience function for creating consistent probe responses.
//
// The status should be a simple string like "ok" or "failed".
// The details map can contain any additional context about the probe result.
// For example:
//
//	NewProbeResponse("ok", map[string]interface{}{
//	    "database": "connected",
//	    "cache_hit_rate": 0.95,
//	})
func NewProbeResponse(status string, details map[string]interface{}) ProbeResponse {
	return ProbeResponse{
		Status:  status,
		Details: details,
	}
}
