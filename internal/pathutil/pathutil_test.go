package pathutil

import "testing"

func TestHasPathPrefix(t *testing.T) {
	tests := []struct {
		path   string
		prefix string
		want   bool
	}{
		// exact match
		{"/api", "/api", true},
		{"/health", "/health", true},

		// subpath match
		{"/api/users", "/api", true},
		{"/api/users/123", "/api", true},
		{"/health/live", "/health", true},

		// boundary violations — must NOT match
		{"/apikeys", "/api", false},
		{"/apiv2/users", "/api", false},
		{"/api-internal", "/api", false},
		{"/healthcheck", "/health", false},

		// unrelated paths
		{"/metrics", "/api", false},
		{"/", "/api", false},
	}

	for _, tc := range tests {
		got := HasPathPrefix(tc.path, tc.prefix)
		if got != tc.want {
			t.Errorf("HasPathPrefix(%q, %q) = %v, want %v", tc.path, tc.prefix, got, tc.want)
		}
	}
}
