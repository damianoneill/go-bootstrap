package http

import (
	"path"
	"strings"
)

// defaultMatcher provides basic path matching capability
type defaultMatcher struct{}

// newMatcher creates a new path matcher
func newMatcher() *defaultMatcher {
	return &defaultMatcher{}
}

// Matches implements PathMatcher
func (m *defaultMatcher) Matches(reqPath string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	// Clean and split the request path
	reqPath = path.Clean("/" + reqPath)
	reqSegments := strings.Split(strings.Trim(reqPath, "/"), "/")

	for _, pattern := range patterns {
		// Clean and split the pattern
		pattern = path.Clean("/" + pattern)
		patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")

		if matchSegments(reqSegments, patternSegments) {
			return true
		}
	}

	return false
}

// matchSegments performs segment-by-segment matching with wildcard support
func matchSegments(reqSegments, patternSegments []string) bool {
	// Handle root path special case
	if len(reqSegments) == 1 && reqSegments[0] == "" && len(patternSegments) == 1 && patternSegments[0] == "" {
		return true
	}

	// Handle trailing wildcard
	if len(patternSegments) > 0 && patternSegments[len(patternSegments)-1] == "*" {
		patternSegments = patternSegments[:len(patternSegments)-1]
		return len(reqSegments) >= len(patternSegments) &&
			matchExactSegments(reqSegments[:len(patternSegments)], patternSegments)
	}

	// Handle segment wildcards and exact matches
	if len(reqSegments) != len(patternSegments) {
		return false
	}

	return matchExactSegments(reqSegments, patternSegments)
}

// matchExactSegments compares segments allowing for wildcards
func matchExactSegments(reqSegments, patternSegments []string) bool {
	for i := range patternSegments {
		if patternSegments[i] != "*" && patternSegments[i] != reqSegments[i] {
			return false
		}
	}
	return true
}
