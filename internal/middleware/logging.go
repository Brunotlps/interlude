// Three responsibilities:
//
//  1. Capture status code — via ResponseWriter wrapper
//  2. Measure duration — time.Now() before, time.Since() after
//  3. Emit structured log — log/slog with key-value fields
//
// Log level by status: 2xx/3xx → Info, 4xx → Warn, 5xx → Error

package middleware

import (
	"interlude/internal/metrics"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type wrappedWriter struct {
	http.ResponseWriter // Embeds the interface
	statusCode          int
}

func (w *wrappedWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code) // Sends to the real Writer
}

func newResponseWriter(w http.ResponseWriter) *wrappedWriter {
	return &wrappedWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // 200 by Default
	}
}

// routeNormalizer matches a raw path to its configured route prefix,
// bounding Prometheus label cardinality to the number of configured routes.
type routeNormalizer struct {
	prefixes []string // sorted longest-first, mirrors router priority
}

func newRouteNormalizer(prefixes []string) *routeNormalizer {
	sorted := make([]string, len(prefixes))
	copy(sorted, prefixes)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i]) > len(sorted[j])
	})
	return &routeNormalizer{prefixes: sorted}
}

func (n *routeNormalizer) normalize(path string) string {
	for _, p := range n.prefixes {
		if strings.HasPrefix(path, p) {
			return p
		}
	}
	return "unknown"
}

func Logging(routePrefixes []string, next http.Handler) http.Handler {
	norm := newRouteNormalizer(routePrefixes)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		metrics.ActiveRequests.Inc()
		defer metrics.ActiveRequests.Dec()

		next.ServeHTTP(rw, r)

		routeLabel := norm.normalize(r.URL.Path)

		metrics.RequestsTotal.WithLabelValues(r.Method, routeLabel, strconv.Itoa(rw.statusCode)).Inc()
		metrics.RequestDuration.WithLabelValues(r.Method, routeLabel).Observe(time.Since(start).Seconds())

		level := slog.LevelInfo
		if rw.statusCode >= 500 {
			level = slog.LevelError
		} else if rw.statusCode >= 400 {
			level = slog.LevelWarn
		}

		slog.Log(r.Context(), level, "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}
