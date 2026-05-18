package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Backend return 200 → healthy
func TestChecker_Probe_Healthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // server.URL → "http://127.0.0.1:PORT" (random port)
	}))
	defer server.Close()

	b := NewBackend(server.URL)
	c := New([]*Backend{b}, time.Minute, 5*time.Second, "/")
	c.probe(context.Background(), b)

	if !b.IsHealthy() {
		t.Error("backend must be healthy after 200 response")
	}
}

// Backend return 500 → unhealthy
func TestChecker_Probe_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // ← 500
	}))
	defer server.Close()

	b := NewBackend(server.URL)
	c := New([]*Backend{b}, time.Minute, 5*time.Second, "/")
	c.probe(context.Background(), b)

	if b.IsHealthy() {
		t.Error("backend must be unhealthy after 500 response")
	}
}

// Inaccessible Backend  → unhealthy
func TestChecker_Probe_Unreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // ← 200
	}))

	b := NewBackend(server.URL)
	c := New([]*Backend{b}, time.Minute, 5*time.Second, "/")
	server.Close()
	c.probe(context.Background(), b)

	if b.IsHealthy() {
		t.Error("backend must be unhealthy inaccessible backend")
	}
}

// Recovery test
func TestChecker_Probe_Recovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	b := NewBackend(server.URL)
	b.SetHealthy(false)
	c := New([]*Backend{b}, time.Minute, 5*time.Second, "/")
	c.probe(context.Background(), b)

	if !b.IsHealthy() {
		t.Error("backend must be healthy after probe")
	}
}
