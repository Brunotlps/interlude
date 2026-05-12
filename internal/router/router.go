package router

import (
	"fmt"
	"interlude/internal/config"
	"interlude/internal/proxy"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

type route struct {
	prefix  string
	handler http.Handler
}

// Gateway core component
// Directly passed to http.ListenAndServe
type Router struct {
	routes []route // sorted by prefix length descending (most specific first)
}

// New creates a Router with routes sorted by prefix length.
// Longer prefixes take priority, so /api/users matches before /api.
func New(cfg *config.Config) (*Router, error) {
	routes := make([]route, 0, len(cfg.Routes))

	for _, r := range cfg.Routes {
		handler, err := proxy.New(r.Backend)
		if err != nil {
			return nil, fmt.Errorf("creating proxy for %s: %w", r.Backend, err)
		}

		routes = append(routes, route{prefix: r.Prefix, handler: handler})
		log.Printf("[ROUTE] %s -> %s", r.Prefix, r.Backend)
	}

	sort.Slice(routes, func(i, j int) bool {
		return len(routes[i].prefix) > len(routes[j].prefix)
	})

	return &Router{routes: routes}, nil
}

// ServeHTTP implements http.Handler.
// Called for every request that arrives at the gateway.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	for _, route := range rt.routes {
		if strings.HasPrefix(r.URL.Path, route.prefix) {
			log.Printf("[REQUEST] %s %s -> routed to %s", r.Method, r.URL.Path, route.prefix)
			route.handler.ServeHTTP(w, r)
			log.Printf("[COMPLETED] %s %s in %v", r.Method, r.URL.Path, time.Since(start))
			return
		}
	}

	log.Printf("[NOT FOUND] %s %s - no matching route", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{"error": "route not found", "path": "%s"}`, r.URL.Path)
}
