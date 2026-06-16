package pathutil

import "strings"

// HasPathPrefix reports whether path starts with the given URL path prefix.
// Unlike strings.HasPrefix, it respects segment boundaries: "/api" matches
// "/api" and "/api/users" but not "/apikeys" or "/apiv2".
func HasPathPrefix(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}
