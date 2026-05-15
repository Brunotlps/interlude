package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "gateway_requests_total", Help: "total number of requests"},
		[]string{"method", "path", "status"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "request time duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	BackendHealth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "gateway_backend_health", Help: "backend health verification"},
		[]string{"backend"},
	)

	CircuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "gateway_circuit_breaker_state", Help: "circuit breaker state verification"},
		[]string{"backend"},
	)

	ActiveRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{Name: "gateway_active_requests", Help: "number of active requests"},
	)
)

func init() {
	prometheus.MustRegister(
		RequestsTotal,
		RequestDuration,
		BackendHealth,
		CircuitBreakerState,
		ActiveRequests,
	)
}
