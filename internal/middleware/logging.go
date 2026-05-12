// Three responsibilities:

//   1. Capture status code — via ResponseWriter wrapper
//   2. Measure duration — time.Now() before, time.Since() after
//   3. Emit structured log — log/slog with key-value fields

// Log level by status: 2xx/3xx → Info, 4xx → Warn, 5xx → Error

package middleware

import (
	"log/slog"
	"net/http"
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

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		next.ServeHTTP(rw, r)

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
