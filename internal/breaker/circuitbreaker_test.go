package breaker

import (
	"testing"
	"time"
)

// helper to not repeat the same verification on many tests
func assertState(t *testing.T, cb *CircuitBreaker, want State) {
	t.Helper()
	currentstate := cb.state.Load()
	if currentstate != int32(want) {
		t.Errorf("state: got %v, want %v", State(currentstate), want)
	}
}

// helper HALF-OPEN setup
func forceHalfOpen(t *testing.T, cb *CircuitBreaker) {
	t.Helper()
	for i := 0; i < int(cb.maxFailures); i++ {
		cb.RecordFailure()
	}
	time.Sleep(cb.recoveryTime + 5*time.Millisecond)
	cb.Allow()
}

// Initial State
func TestNew_InitialState(t *testing.T) {
	cb := New("test", 3, time.Minute)
	if cb == nil {
		t.Fatal("CircuitBreaker.New() failed")
	}
	assertState(t, cb, StateClosed)
	if !cb.Allow() {
		t.Error("Allow() must return true when the state is Closed")
	}
}

// Failure transitions - table-driven
func TestRecordFailure_StateTransitions(t *testing.T) {
	tests := []struct {
		name      string
		failures  int
		wantState State
		wantAllow bool
	}{
		{"below threshold stays closed", 2, StateClosed, true},
		{"at threshold opens", 3, StateOpen, false},
		{"above threshold stays open", 5, StateOpen, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := New("test", 3, time.Minute)
			for i := 0; i < tt.failures; i++ {
				cb.RecordFailure()
			}
			assertState(t, cb, tt.wantState)
			got := cb.Allow()
			if got != tt.wantAllow {
				t.Errorf("Allow() = %v, want %v", got, tt.wantAllow)
			}
		})
	}
}

// OPEN -> HALF-OPEN after recovery timeout
func TestAllow_TransitionsToHalfOpenAfterRecovery(t *testing.T) {
	cb := New("test", 3, 10*time.Millisecond)
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	assertState(t, cb, StateOpen)
	if cb.Allow() {
		t.Error("Allow() must return false while OPEN before recovery")
	}
	time.Sleep(15 * time.Millisecond)
	if !cb.Allow() {
		t.Error("Allow() must return true after recovery timeout")
	}
	assertState(t, cb, StateHalfOpen)
}

// HALF-OPEN → OPEN in case of new failure
func TestRecordFailure_ReopensFromHalfOpen(t *testing.T) {
	cb := New("test", 3, 10*time.Millisecond)
	forceHalfOpen(t, cb)
	assertState(t, cb, StateHalfOpen)
	cb.RecordFailure()
	assertState(t, cb, StateOpen)
}

// HALF-OPEN → CLOSED in case of success
func TestRecordSuccess_ClosesFromHalfOpen(t *testing.T) {
	cb := New("test", 3, 10*time.Millisecond)
	forceHalfOpen(t, cb)
	cb.RecordSuccess()
	assertState(t, cb, StateClosed)
	if !cb.Allow() {
		t.Error("Allow() must return true after closes from half-open")
	}
}

// RecordSuccess out of HALF-OPEN its no-op
func TestRecordSuccess_NoopWhenClosed(t *testing.T) {
	cb := New("test", 3, 10*time.Millisecond)
	cb.RecordSuccess()
	assertState(t, cb, StateClosed)

}
