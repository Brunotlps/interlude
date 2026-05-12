package middleware

import (
	"fmt"
	"interlude/internal/config"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64
	burst   float64
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{tokens: rl.burst, lastRefill: time.Now()}
		rl.buckets[ip] = b
	}

	elapsed := time.Since(b.lastRefill).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.lastRefill = time.Now()

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (rl *RateLimiter) startCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			rl.mu.Lock()
			for ip, b := range rl.buckets {
				if time.Since(b.lastRefill) > interval*2 {
					delete(rl.buckets, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
}

func RateLimit(cfg config.RateLimitConfig) func(http.Handler) http.Handler {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    cfg.RequestsPerSecond,
		burst:   cfg.Burst,
	}
	rl.startCleanup(5 * time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			if !rl.allow(ip) {
				slog.Warn("rate limit exceeded", "ip", ip, "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = fmt.Fprintf(w, `{"error": "rate limit exceeded"}`)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
