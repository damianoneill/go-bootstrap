// pkg/adapter/http/matcher_test.go
package http

import (
	"testing"
)

func TestMatcher(t *testing.T) {
	matcher := newMatcher()

	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{
			name:     "empty patterns",
			path:     "/test",
			patterns: nil,
			want:     false,
		},
		{
			name:     "exact match",
			path:     "/test",
			patterns: []string{"/test"},
			want:     true,
		},
		{
			name:     "no match",
			path:     "/test",
			patterns: []string{"/other"},
			want:     false,
		},
		{
			name:     "wildcard match",
			path:     "/api/v1/users",
			patterns: []string{"/api/*"},
			want:     true,
		},
		{
			name:     "wildcard no match",
			path:     "/api/v1/users",
			patterns: []string{"/other/*"},
			want:     false,
		},
		{
			name:     "multiple patterns with match",
			path:     "/test",
			patterns: []string{"/other", "/test", "/another"},
			want:     true,
		},
		{
			name:     "path cleaning - trailing slash",
			path:     "/test/",
			patterns: []string{"/test"},
			want:     true,
		},
		{
			name:     "path cleaning - double slash",
			path:     "/test//path",
			patterns: []string{"/test/path"},
			want:     true,
		},
		{
			name:     "path cleaning - dot segments",
			path:     "/test/./path",
			patterns: []string{"/test/path"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.Matches(tt.path, tt.patterns)
			if got != tt.want {
				t.Errorf("Matches(%q, %v) = %v, want %v",
					tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestMatcherEdgeCases(t *testing.T) {
	matcher := newMatcher()

	tests := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{
			name:     "empty path",
			path:     "",
			patterns: []string{"/"},
			want:     true,
		},
		{
			name:     "root path",
			path:     "/",
			patterns: []string{"/"},
			want:     true,
		},
		{
			name: "complex wildcard patterns",
			path: "/api/v1/users/123/profile",
			patterns: []string{
				"/api/v1/users/*/profile",
				"/api/v2/*",
			},
			want: true,
		},
		{
			name:     "case sensitivity",
			path:     "/API/users",
			patterns: []string{"/api/users"},
			want:     false,
		},
		{
			name:     "encoded characters",
			path:     "/test%20path",
			patterns: []string{"/test path"},
			want:     false,
		},
		{
			name:     "segment wildcard",
			path:     "/api/v1/users/123/profile",
			patterns: []string{"/api/v1/users/*/profile"},
			want:     true,
		},
		{
			name:     "segment wildcard no match",
			path:     "/api/v1/users/123/settings",
			patterns: []string{"/api/v1/users/*/profile"},
			want:     false,
		},
		{
			name:     "multiple segment wildcards",
			path:     "/api/v1/users/123/posts/456",
			patterns: []string{"/api/*/users/*/posts/*"},
			want:     true,
		},
		{
			name:     "trailing wildcard only matches remaining segments",
			path:     "/api/v1/users",
			patterns: []string{"/api/*"},
			want:     true,
		},
		{
			name:     "root path with trailing slash",
			path:     "/",
			patterns: []string{"/"},
			want:     true,
		},
		{
			name:     "empty paths",
			path:     "",
			patterns: []string{""},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.Matches(tt.path, tt.patterns)
			if got != tt.want {
				t.Errorf("Matches(%q, %v) = %v, want %v",
					tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}
