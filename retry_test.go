package retry

import (
	"context"
	"math"
	"testing"
	"time"
	"unsafe"

	"errors"
	"testing/synctest"

	"github.com/nalgeon/be"
)

var errAgain = errors.New("some retriable error")

func TestNext_EternalRetry(t *testing.T) {
	expectedRetries := []uint{1, 2, 3}
	sut := NewExponentialBackoff(
		5*time.Millisecond,
		math.MaxUint,
		2*time.Millisecond,
	)

	for _, expected := range expectedRetries {
		_, err := sut.Next()

		be.Err(t, err, nil)
		be.Equal(t, sut.RetriesCount(), expected)
	}
}

func TestReset(t *testing.T) {
	sut := NewExponentialBackoff(
		5*time.Millisecond,
		3,
		2*time.Millisecond,
	)

	for range 3 {
		_, err := sut.Next()
		be.Err(t, err, nil)
	}
	sut.Reset()

	be.Equal(t, sut.RetriesCount(), uint(0))
	be.Equal(t, sut.reachedMaxDelay, false)
}

func TestWithRetry(t *testing.T) {
	tests := []struct {
		name     string
		results  []error
		retries  uint
		expected error
	}{
		{
			name:     "First request succeeded",
			results:  []error{nil},
			retries:  0,
			expected: nil,
		},
		{
			name:     "First request failed, second succeeded",
			results:  []error{errAgain, nil},
			retries:  1,
			expected: nil,
		},
		{
			name:     "All attempts failed",
			results:  []error{errAgain, errAgain, errAgain, errAgain},
			retries:  3,
			expected: ErrMaxRetry,
		},
	}

	for _, tt := range tests {
		synctest.Test(t, func(t *testing.T) {
			ctx := t.Context()

			sut := NewExponentialBackoff(
				5*time.Millisecond,
				3,
				2*time.Millisecond,
			)
			var err error

			go func() {
				i := -1
				err = sut.WithRetry(ctx, func() error {
					i++
					return tt.results[i]
				})
			}()

			time.Sleep(16 * time.Millisecond)
			synctest.Wait()

			be.Err(t, err, tt.expected)
			be.Equal(t, sut.RetriesCount(), tt.retries)
		})
	}
}

func TestWithRetry_Canceled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		sut := NewExponentialBackoff(
			5*time.Millisecond,
			3,
			2*time.Millisecond,
		)
		var err error

		go func() {
			err = sut.WithRetry(ctx, func() error {
				return errAgain
			})
		}()
		cancel()

		synctest.Wait()

		be.Err(t, err, context.Canceled)
	})
}

func TestWithRetry_Overflow(t *testing.T) {
	sut := NewExponentialBackoff(
		5*time.Minute,
		math.MaxUint,
		2*time.Millisecond,
	)

	// The counter is 64 bits wide, so after 65 iterations it is guaranteed to overflow
	// since we use powers of 2 for the delay.
	const counterBitSize = 8*unsafe.Sizeof(ExponentialBackoff{}.retriesCount) + 1

	for range counterBitSize {
		_, err := sut.Next()
		be.Err(t, err, nil)
	}
}
