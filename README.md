# interlude

[![CI](https://github.com/Brunotlps/interlude/actions/workflows/ci.yaml/badge.svg)](https://github.com/Brunotlps/interlude/actions/workflows/ci.yaml)
[![Go](https://img.shields.io/badge/go-1.26-blue)](https://go.dev/)

A production-grade HTTP reverse proxy and API gateway built from scratch in Go — no frameworks, stdlib-first.

Built as a learning project to deeply understand how gateway infrastructure works: concurrency primitives, resilience patterns, observability, and containerization.

## Features

- **Reverse proxy** — URL-prefix-based routing to multiple backend pools
- **Load balancing** — round-robin with atomic counters, skips unhealthy backends
- **Health checking** — per-backend goroutine with configurable interval and timeout
- **Circuit breaker** — three-state machine (Closed → Open → Half-Open) using lock-free CAS
- **Retry with backoff** — exponential backoff with jitter, context-aware cancellation
- **Rate limiting** — token bucket per gateway instance
- **Authentication** — API key middleware via `X-API-Key` header
- **Structured logging** — JSON output via `log/slog`
- **Prometheus metrics** — request counters, latency histograms, backend health and circuit breaker state gauges
- **Docker** — multi-stage build, static binary, ~17MB final image on `scratch`

## Architecture

```
Request
  → Logging middleware
  → Rate limiter middleware
  → Auth middleware
  → Router (prefix match, longest first)
    → Load balancer (picks healthy backend)
    → Circuit breaker (allow?)
      YES → Proxy request with timeout
            → Success: RecordSuccess, return response
            → Failure: RecordFailure, retry with next backend
      NO  → 503 immediately

Background: health checker goroutine per backend (ticker + context)
```

## Quick Start

**With Docker Compose** (gateway + 3 backends + Prometheus + Grafana):

```bash
git clone https://github.com/Brunotlps/interlude.git
cd interlude
docker-compose up
```

| Service    | URL                           |
|------------|-------------------------------|
| Gateway    | http://localhost:8080         |
| Metrics    | http://localhost:9091/metrics |
| Prometheus | http://localhost:9090         |
| Grafana    | http://localhost:3000         |

Make a request:

```bash
curl -H "X-API-Key: key-abc-123" http://localhost:8080/api/users
```

**Run locally** (requires Go 1.26+):

```bash
go run ./cmd/gateway
```

## Configuration

The gateway reads `config.yaml` by default. Override with `CONFIG_PATH=/path/to/config.yaml`.

```yaml
server:
  port: 8080
  metrics_port: 9091

auth:
  api_keys:
    - "your-api-key"

rate_limit:
  requests_per_second: 10
  burst: 20

routes:
  - prefix: "/api/users"
    backends:
      - "http://backend1:3001"
      - "http://backend2:3002"

health_check:
  interval: 10s
  timeout: 5s
  path: "/health"

circuit_breaker:
  max_failures: 5
  recovery_timeout: 30s

retry:
  max_attempts: 3
```

| Field | Description |
|-------|-------------|
| `server.port` | Gateway proxy port |
| `server.metrics_port` | Prometheus scrape port (separate from proxy traffic) |
| `auth.api_keys` | Accepted API keys for `X-API-Key` header |
| `rate_limit.requests_per_second` | Token bucket refill rate |
| `rate_limit.burst` | Maximum burst size |
| `routes[].prefix` | URL prefix to match (longest prefix wins) |
| `routes[].backends` | Backend pool URLs for this route |
| `health_check.interval` | How often to probe each backend |
| `health_check.path` | Path used for health probe requests |
| `circuit_breaker.max_failures` | Consecutive failures before opening the circuit |
| `circuit_breaker.recovery_timeout` | Time in Open state before attempting Half-Open |
| `retry.max_attempts` | Max attempts per request including the first |

## Metrics

All metrics are exposed at `:9091/metrics` in Prometheus text format.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `gateway_requests_total` | Counter | `method`, `path`, `status` | Total proxied requests |
| `gateway_request_duration_seconds` | Histogram | `method`, `path` | Request latency |
| `gateway_backend_health` | Gauge | `backend` | Backend health (1=healthy, 0=unhealthy) |
| `gateway_circuit_breaker_state` | Gauge | `backend` | Circuit state (0=closed, 1=open, 2=half-open) |
| `gateway_active_requests` | Gauge | — | Requests currently in flight |

## Development

**Run tests:**

```bash
go test -race ./...
```

**Build binary:**

```bash
CGO_ENABLED=0 go build -o gateway ./cmd/gateway
```

**Build Docker image:**

```bash
docker build -t interlude .
```

## Project Structure

```
.
├── cmd/gateway/main.go          # Entrypoint — wiring only
├── config.yaml                  # Default configuration
├── config.docker.yaml           # Configuration for docker-compose
├── Dockerfile                   # Multi-stage build (builder → scratch)
├── docker-compose.yaml          # Full stack: gateway, backends, Prometheus, Grafana
├── prometheus.yml               # Prometheus scrape config
└── internal/
    ├── balancer/                # Round-robin load balancer
    ├── breaker/                 # Circuit breaker (3-state CAS)
    ├── config/                  # YAML config loader
    ├── health/                  # Per-backend health checker goroutines
    ├── metrics/                 # Prometheus metric definitions
    ├── middleware/              # Logging, rate limiting, auth
    ├── proxy/                   # HTTP reverse proxy client
    ├── retry/                   # Exponential backoff with jitter
    └── router/                  # Request routing and resilience orchestration
```
