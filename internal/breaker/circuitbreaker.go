package breaker

import (
	"interlude/internal/metrics"
	"sync/atomic"
	"time"
)

type State int32

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	name         string
	state        atomic.Int32
	failures     atomic.Int32
	maxFailures  int32
	openedAt     atomic.Int64
	recoveryTime time.Duration
}

func New(name string, maxFailures int32, recoveryTime time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:         name,
		maxFailures:  maxFailures,
		recoveryTime: recoveryTime,
	}
}

func (cb *CircuitBreaker) Allow() bool {

	state := State(cb.state.Load())

	switch state {
	case StateClosed:
		return true
	case StateHalfOpen:
		return false
	case StateOpen:
		openedAt := time.Unix(0, cb.openedAt.Load()) // time.Unix(0, nanos) converts the int64 (nanoseconds) back to time.Time to compare with time.Since
		if time.Since(openedAt) < cb.recoveryTime {
			return false
		}
		if cb.state.CompareAndSwap(int32(StateOpen), int32(StateHalfOpen)) {
			metrics.CircuitBreakerState.WithLabelValues(cb.name).Set(float64(StateHalfOpen))
			return true
		}
	}
	return false
}

func (cb *CircuitBreaker) RecordFailure() {

	if cb.state.CompareAndSwap(int32(StateHalfOpen), int32(StateOpen)) {
		metrics.CircuitBreakerState.WithLabelValues(cb.name).Set(float64(1))
		cb.openedAt.Store(time.Now().UnixNano())
		cb.failures.Store(0)
		return
	}

	failures := cb.failures.Add(1)
	if failures >= cb.maxFailures {
		if cb.state.CompareAndSwap(int32(StateClosed), int32(StateOpen)) {
			metrics.CircuitBreakerState.WithLabelValues(cb.name).Set(float64(1))
			cb.openedAt.Store(time.Now().UnixNano())
			cb.failures.Store(0)
		}
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	if cb.state.CompareAndSwap(int32(StateHalfOpen), int32(StateClosed)) {
		metrics.CircuitBreakerState.WithLabelValues(cb.name).Set(float64(0))
		cb.failures.Store(0)
	}
}
