package retry

import (
	"context"
	"math"
	"math/rand/v2"
	"time"

	"errors"
)

type Retriable = func() error

var ErrMaxRetry = errors.New("maximum retry attempts count reached")

// ExponentialBackoff implements the exponential backoff algorithm with jitter.
// Applies the full jitter algorithm to ensure maximum spread of values.
type ExponentialBackoff struct {
	// maxDelay is the maximum delay between retries.
	maxDelay time.Duration

	// retriesCount is the number of retries performed so far.
	retriesCount uint

	// maxRetryCount is the maximum number of retries allowed.
	maxRetryCount uint

	// baseDelay is the base delay between retries.
	baseDelay time.Duration

	reachedMaxDelay bool
}

// NewExponentialBackoff creates a new [ExponentialBackoff] instance.
func NewExponentialBackoff(
	maxDelay time.Duration,
	maxRetryCount uint,
	baseDelay time.Duration,
) *ExponentialBackoff {
	return &ExponentialBackoff{
		maxDelay:      maxDelay,
		maxRetryCount: maxRetryCount,
		baseDelay:     baseDelay,
	}
}

// Next returns the next timeout duration before the following retry.
// Returns [ErrMaxRetry] when the maximum number of retry attempts is reached.
func (e *ExponentialBackoff) Next() (time.Duration, error) {
	if e.retriesCount+1 > e.maxRetryCount {
		return 0, ErrMaxRetry
	}

	e.retriesCount++

	if e.reachedMaxDelay {
		return fullJitter(e.maxDelay), nil
	}

	delay := e.baseDelay * time.Duration(math.Pow(2, float64(e.retriesCount)))
	if delay >= e.maxDelay {
		e.reachedMaxDelay = true

		delay = e.maxDelay
	}

	return fullJitter(delay), nil
}

func fullJitter(delay time.Duration) time.Duration {
	return time.Duration(rand.Int64N(int64(delay)))
}

// Reset resets the backoff to its initial state so that it can be reused.
func (e *ExponentialBackoff) Reset() {
	e.retriesCount = 0
	e.reachedMaxDelay = false
}

// WithRetry executes the given action, retrying it on error according to the backoff policy.
// The action is called immediately on the first attempt if the context is not already cancelled,
// and after each exponential backoff delay on subsequent attempts.
// Returns nil if the action succeeds, [ErrMaxRetry] if the maximum number of attempts is exhausted,
// or ctx.Err() if the context is cancelled while waiting between retries.
// Each call to WithRetry must use either a separate [ExponentialBackoff] instance or call [ExponentialBackoff.Reset] beforehand.
func (e *ExponentialBackoff) WithRetry(ctx context.Context, action Retriable) error {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-timer.C:
			err := action()
			if err == nil {
				return nil
			}

			timeout, err := e.Next()
			if err != nil {
				return err
			}

			timer.Reset(timeout)
		}
	}
}

// RetriesCount returns the number of retries performed so far.
func (e *ExponentialBackoff) RetriesCount() uint {
	return e.retriesCount
}
