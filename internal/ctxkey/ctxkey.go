// Package ctxkey defines context keys shared across packages.
// Using a named struct type (instead of a plain string) prevents collisions
// with keys from other packages that might use the same string value.
package ctxkey

// RouteLabelKey is the context key for the matched route prefix.
// The logging middleware stores a *string under this key before calling
// the next handler; the router writes the matched prefix through that pointer.
type RouteLabelKey struct{}
