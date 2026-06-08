package balancer

import (
	"errors"
	"interlude/internal/health"
	"sync/atomic"
)

// atomic.Uint64 -> increment and read in a single CPU instruction

type Balancer struct {
	backends []*health.Backend
	counter  atomic.Uint64
}

// Constructor
func New(backends []*health.Backend) *Balancer {
	return &Balancer{backends: backends}
}

func (b *Balancer) Next() (*health.Backend, error) {
	n := uint64(len(b.backends))
	start := b.counter.Add(1)
	for i := range n {
		idx := (start + i) % n
		if b.backends[idx].IsHealthy() {
			return b.backends[idx], nil
		}
	}
	return nil, errors.New("no healthy backend available")
}
