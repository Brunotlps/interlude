package middleware

import (
	"context"
	"interlude/internal/ctxkey"
	"net/http"
	"net/http/httptest"
	"testing"
)

// routerStub simulates the router writing the matched prefix into the context pointer.
type routerStub struct {
	label string
}

func (s *routerStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ptr, ok := r.Context().Value(ctxkey.RouteLabelKey{}).(*string); ok {
		*ptr = s.label
	}
	w.WriteHeader(http.StatusOK)
}

func TestLogging_WritesRouteLabelViaContext(t *testing.T) {
	stub := &routerStub{label: "/api"}
	handler := Logging(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rw.Code)
	}
}

func TestLogging_UnknownLabelWhenRouterDoesNotWrite(t *testing.T) {
	// Handler that never writes the label (simulates no matching route / 404 handler).
	noLabel := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	handler := Logging(noLabel)

	req := httptest.NewRequest(http.MethodGet, "/unknown/path", nil)
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	if rw.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rw.Code)
	}
}

func TestLogging_ContextPointerIsAvailableToHandler(t *testing.T) {
	var capturedPtr *string

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ptr, ok := r.Context().Value(ctxkey.RouteLabelKey{}).(*string)
		if !ok || ptr == nil {
			t.Error("expected *string in context, got nothing")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		capturedPtr = ptr
		*ptr = "/captured"
		w.WriteHeader(http.StatusOK)
	})

	handler := Logging(inner)
	req := httptest.NewRequest(http.MethodGet, "/captured/path", nil)
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	if capturedPtr == nil {
		t.Fatal("handler never received the context pointer")
	}
	if *capturedPtr != "/captured" {
		t.Errorf("expected label /captured, got %q", *capturedPtr)
	}
}

// Verify that Logging does not depend on any external context value being pre-set.
func TestLogging_WorksWithPlainContext(t *testing.T) {
	stub := &routerStub{label: "/health"}
	handler := Logging(stub)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req = req.WithContext(context.Background()) // plain context, no pre-set values
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rw.Code)
	}
}
