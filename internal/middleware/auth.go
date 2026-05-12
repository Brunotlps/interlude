package middleware

import (
	"log/slog"
	"net/http"
)

func Auth(validKeys map[string]struct{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")

			if _, ok := validKeys[key]; !ok {
				slog.Warn("unauthorized request",
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
				return
			}

			next.ServeHTTP(w, r)

		})
	}
}
