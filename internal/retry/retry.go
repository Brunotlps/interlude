package retry

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

func Do(ctx context.Context, maxAttempts int, fn func() error) error {
	const baseDelay = 100 * time.Millisecond

	var err error
	for i := 0; i < maxAttempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if i == maxAttempts-1 {
			break
		}

		backoff := baseDelay * time.Duration(1<<uint(i))
		jitter := time.Duration(rand.Int63n(int64(baseDelay)))
		wait := backoff + jitter

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("all %d attempts failed: %w", maxAttempts, err)
}
