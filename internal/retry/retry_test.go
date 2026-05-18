package retry

import (
	"context"
	"errors"
	"testing"
)

func TestDo_SuccessOnFirstAttempt(t *testing.T) {
	calls := 0
	fn := func() error {
		calls++
		return nil
	}

	err := Do(context.Background(), 3, fn)
	if err != nil {
		t.Errorf("Do() returned error: %v", err)
	}
	if calls != 1 {
		t.Errorf("fn called %d times, want 1", calls)
	}
}

func TestDo_AllAttemptsFail(t *testing.T) {
	calls := 0
	fn := func() error {
		calls++
		return errors.New("fail")
	}
	err := Do(context.Background(), 1, fn)
	if err == nil {
		t.Error("Do() must return error when all attempts fail")
	}
	if calls != 1 {
		t.Errorf("calls = %v, wanted: %v", calls, 1)
	}
}

func TestDo_SuccessAfterRetry(t *testing.T) {
	calls := 0
	fn := func() error {
		calls++
		if calls >= 2 {
			return nil
		}
		return errors.New("fail")
	}
	err := Do(context.Background(), 3, fn)
	if err != nil {
		t.Errorf("Do() returned error: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %v, wanted: %v", calls, 2)
	}
}

func TestDo_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fn := func() error { return errors.New("fail") }

	err := Do(ctx, 2, fn)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
}
