package health

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// shared between health checker and load balancer
type Backend struct {
	URL     string
	healthy atomic.Bool // atomic for primitive types, mutex for structures composites
}

func (b *Backend) IsHealthy() bool   { return b.healthy.Load() }
func (b *Backend) SetHealthy(v bool) { b.healthy.Store(v) }

func NewBackend(url string) *Backend {
	b := &Backend{URL: url}
	b.healthy.Store(true) // starts healthy
	return b
}

type Checker struct {
	backends []*Backend
	interval time.Duration
	client   *http.Client
	path     string
}

func (c *Checker) Start(ctx context.Context) {
	for _, b := range c.backends {
		go c.run(ctx, b)
	}
}

func (c *Checker) run(ctx context.Context, b *Backend) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	c.probe(ctx, b)

	for {
		select {
		case <-ticker.C:
			c.probe(ctx, b)
		case <-ctx.Done():
			return
		}
	}
}

func (c *Checker) probe(ctx context.Context, b *Backend) {

	// build the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.URL+c.path, nil)
	if err != nil {
		b.SetHealthy(false)
		slog.Warn("health check: failed to build request", "backend", b.URL, "err", err)
		return
	}

	// sends the request
	resp, err := c.client.Do(req)
	if err != nil {
		b.SetHealthy(false)
		slog.Warn("health check: unreachable", "backend", b.URL, "err", err)
		return
	}

	defer resp.Body.Close()

	wasHealthy := b.IsHealthy()
	isHealthy := resp.StatusCode < 500
	b.SetHealthy(isHealthy)

	if wasHealthy != isHealthy {
		if isHealthy {
			slog.Info("health check: backend recovered", "backend", b.URL)
		} else {
			slog.Warn("health check: backend unhealthy", "backend", b.URL, "status", resp.StatusCode)
		}
	}
}

func New(backends []*Backend, interval, timeout time.Duration, path string) *Checker {
	return &Checker{
		backends: backends,
		interval: interval,
		client:   &http.Client{Timeout: timeout}, // http.Client avoids goroutine leak
		path:     path,
	}
}
