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

	// Clean the request path
	reqPath = path.Clean("/" + reqPath)

	for _, pattern := range patterns {
		pattern = path.Clean("/" + pattern)

		// Direct match
		if pattern == reqPath {
			return true
		}

		// Wildcard matching
		if strings.HasSuffix(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(reqPath, prefix) {
				return true
			}
		}
	}

	return false
}
