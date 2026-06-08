package balancer

import (
	"interlude/internal/health"
	"testing"
)

func makeBackend(url string, healthy bool) *health.Backend {
	b := health.NewBackend(url)
	b.SetHealthy(healthy)
	return b
}

func TestNext_RoundRobin(t *testing.T) {
	backends := []*health.Backend{
		makeBackend("http://a:1", true),
		makeBackend("http://b:2", true),
		makeBackend("http://c:3", true),
	}
	bal := New(backends)

	seen := make(map[string]int)
	for range 9 {
		b, err := bal.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		seen[b.URL]++
	}

	for _, b := range backends {
		if seen[b.URL] != 3 {
			t.Errorf("backend %s called %d times, want 3", b.URL, seen[b.URL])
		}
	}
}

func TestNext_SkipsUnhealthyBackend(t *testing.T) {
	backends := []*health.Backend{
		makeBackend("http://a:1", false),
		makeBackend("http://b:2", true),
		makeBackend("http://c:3", false),
	}
	bal := New(backends)

	for range 5 {
		b, err := bal.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.URL != "http://b:2" {
			t.Errorf("expected healthy backend http://b:2, got %s", b.URL)
		}
	}
}

func TestNext_AllUnhealthy_ReturnsError(t *testing.T) {
	backends := []*health.Backend{
		makeBackend("http://a:1", false),
		makeBackend("http://b:2", false),
	}
	bal := New(backends)

	_, err := bal.Next()
	if err == nil {
		t.Fatal("expected error when all backends unhealthy, got nil")
	}
}

func TestNext_AllUnhealthy_CounterAdvancesOnce(t *testing.T) {
	backends := []*health.Backend{
		makeBackend("http://a:1", false),
		makeBackend("http://b:2", false),
		makeBackend("http://c:3", false),
	}
	bal := New(backends)

	calls := 3
	for range calls {
		bal.Next() //nolint:errcheck
	}

	// counter should be exactly `calls` — one increment per Next() call
	got := bal.counter.Load()
	if got != uint64(calls) {
		t.Errorf("counter = %d after %d Next() calls with all backends unhealthy, want %d", got, calls, calls)
	}
}
