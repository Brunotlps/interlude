package middleware

import "testing"

func TestRouteNormalizer_Normalize(t *testing.T) {
	prefixes := []string{"/api", "/api/users", "/health"}
	norm := newRouteNormalizer(prefixes)

	tests := []struct {
		path string
		want string
	}{
		{"/api/users/42", "/api/users"},
		{"/api/users/99", "/api/users"},
		{"/api/users", "/api/users"},
		{"/api/orders/1", "/api"},
		{"/api", "/api"},
		{"/health", "/health"},
		{"/unknown/path", "unknown"},
		{"/", "unknown"},
	}

	for _, tc := range tests {
		got := norm.normalize(tc.path)
		if got != tc.want {
			t.Errorf("normalize(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestRouteNormalizer_LongestPrefixWins(t *testing.T) {
	// /api/users must win over /api for paths under /api/users
	norm := newRouteNormalizer([]string{"/api", "/api/users"})

	if got := norm.normalize("/api/users/123"); got != "/api/users" {
		t.Errorf("expected longest prefix /api/users, got %q", got)
	}
}

func TestRouteNormalizer_EmptyPrefixes(t *testing.T) {
	norm := newRouteNormalizer([]string{})
	if got := norm.normalize("/anything"); got != "unknown" {
		t.Errorf("expected unknown for empty prefix list, got %q", got)
	}
}
