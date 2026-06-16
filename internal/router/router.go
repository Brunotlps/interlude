package router

import (
	"context"
	"fmt"
	"interlude/internal/balancer"
	"interlude/internal/breaker"
	"interlude/internal/config"
	"interlude/internal/health"
	"interlude/internal/pathutil"
	"interlude/internal/proxy"
	"io"
	"log/slog"
	"net/http"
	"sort"
)

type route struct {
	prefix   string
	balancer *balancer.Balancer
	proxies  map[string]*proxy.Proxy
	breakers map[string]*breaker.CircuitBreaker
}

// Gateway core component
// Directly passed to http.ListenAndServe
type Router struct {
	routes      []route // sorted by prefix length descending (most specific first)
	maxAttempts int
}

// New creates a Router with routes sorted by prefix length.
// Longer prefixes take priority, so /api/users matches before /api.
func New(ctx context.Context, cfg *config.Config) (*Router, error) {
	routes := make([]route, 0, len(cfg.Routes))
	allBackends := make([]*health.Backend, 0)

	for _, r := range cfg.Routes {
		backends := make([]*health.Backend, 0, len(r.Backends))
		proxies := make(map[string]*proxy.Proxy, len(r.Backends))
		breakers := make(map[string]*breaker.CircuitBreaker, len(r.Backends))

		for _, url := range r.Backends {
			b := health.NewBackend(url)
			backends = append(backends, b)
			allBackends = append(allBackends, b)

			p, err := proxy.New(url)
			if err != nil {
				return nil, fmt.Errorf("creating proxy for %s: %w", url, err)
			}
			proxies[url] = p
			breakers[url] = breaker.New(url, cfg.CircuitBreaker.MaxFailures, cfg.CircuitBreaker.RecoveryTimeout)
		}
		bal := balancer.New(backends)
		routes = append(routes, route{prefix: r.Prefix, balancer: bal, proxies: proxies, breakers: breakers})
		slog.Info("route registered", "prefix", r.Prefix, "backends", len(backends))
	}

	checker := health.New(allBackends, cfg.HealthCheck.Interval, cfg.HealthCheck.Timeout, cfg.HealthCheck.Path)
	checker.Start(ctx)
	sort.Slice(routes, func(i, j int) bool {
		return len(routes[i].prefix) > len(routes[j].prefix)
	})

	return &Router{routes: routes, maxAttempts: cfg.Retry.MaxAttempts}, nil
}

// ServeHTTP implements http.Handler.
// Called for every request that arrives at the gateway.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, ro := range rt.routes {
		if !pathutil.HasPathPrefix(r.URL.Path, ro.prefix) {
			continue
		}

		var resp *http.Response

		var reachedBackend bool

		for i := 0; i < rt.maxAttempts; i++ {
			backend, err := ro.balancer.Next()
			if err != nil {
				break
			}

			cb := ro.breakers[backend.URL]
			if !cb.Allow() {
				slog.Warn("circuit breaker open", "backend", backend.URL)
				continue
			}

			reachedBackend = true
			resp, err = ro.proxies[backend.URL].Do(r)
			if err != nil {
				cb.RecordFailure()
				slog.Warn("backend request failed", "backend", backend.URL, "err", err)
				continue
			}

			cb.RecordSuccess()
			break
		}

		if resp == nil {
			w.Header().Set("Content-Type", "application/json")
			if reachedBackend {
				w.WriteHeader(http.StatusBadGateway)
				fmt.Fprintf(w, `{"error": "backend unavailable"}`)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprintf(w, `{"error": "no backends available"}`)
			}
			return
		}

		defer resp.Body.Close()
		for k, v := range resp.Header {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return
	}

	slog.Warn("no matching route", "method", r.Method, "path", r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{"error": "route not found", "path": "%s"}`, r.URL.Path)
}
