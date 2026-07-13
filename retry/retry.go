package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Policy controls how many times to retry and how long to wait between attempts.
type Policy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// DefaultPolicy returns the project defaults (3 attempts, 1s→30s backoff).
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:    3,
		InitialBackoff: time.Second,
		MaxBackoff:     30 * time.Second,
	}
}

type transientError struct {
	err        error
	retryAfter time.Duration
}

func (e *transientError) Error() string { return e.err.Error() }
func (e *transientError) Unwrap() error { return e.err }

// Transient marks err as retryable. Optional retryAfter is honored by Do when > 0.
func Transient(err error, retryAfter ...time.Duration) error {
	if err == nil {
		return nil
	}
	var after time.Duration
	if len(retryAfter) > 0 {
		after = retryAfter[0]
	}
	var te *transientError
	if errors.As(err, &te) {
		if after > te.retryAfter {
			te.retryAfter = after
		}
		return err
	}
	return &transientError{err: err, retryAfter: after}
}

// IsTransient reports whether err (or a wrapped cause) is marked retryable,
// or is a common network timeout / temporary error. context.Canceled is never transient.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	var te *transientError
	if errors.As(err, &te) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

// RetryAfter returns a suggested wait from a Transient-wrapped error, or 0.
func RetryAfter(err error) time.Duration {
	var te *transientError
	if errors.As(err, &te) {
		return te.retryAfter
	}
	return 0
}

// ParseRetryAfterHeader parses an HTTP Retry-After header (seconds or HTTP-date).
func ParseRetryAfterHeader(h http.Header) time.Duration {
	if h == nil {
		return 0
	}
	raw := strings.TrimSpace(h.Get("Retry-After"))
	if raw == "" {
		return 0
	}
	if secs, err := strconv.Atoi(raw); err == nil {
		if secs <= 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(raw); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

// HTTPStatusTransient reports whether status is a typical retryable HTTP code.
func HTTPStatusTransient(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusConflict, http.StatusTooManyRequests:
		return true
	default:
		return status >= 500
	}
}

// Backoff returns the wait for the given zero-based failed attempt index,
// combining exponential backoff, optional Retry-After, and jitter.
func Backoff(p Policy, attempt int, retryAfter time.Duration) time.Duration {
	initial := p.InitialBackoff
	if initial <= 0 {
		initial = time.Second
	}
	max := p.MaxBackoff
	if max <= 0 {
		max = 30 * time.Second
	}
	exp := float64(initial) * math.Pow(2, float64(attempt))
	computed := time.Duration(exp)
	if computed > max {
		computed = max
	}
	wait := computed
	if retryAfter > wait {
		wait = retryAfter
	}
	if wait > max {
		wait = max
	}
	if wait <= 0 {
		return 0
	}
	// Up to 20% jitter.
	jitter := time.Duration(float64(wait) * 0.2 * rand.Float64())
	return wait + jitter
}

// OnRetry is called before sleeping when a transient error will be retried.
// attempt is 1-based for the upcoming retry (after the failed try).
type OnRetry func(attempt, maxAttempts int, wait time.Duration, err error)

// Do runs fn until it succeeds, returns a non-transient error, or exhausts MaxAttempts.
func Do[T any](ctx context.Context, p Policy, fn func() (T, error), onRetry OnRetry) (T, error) {
	var zero T
	max := p.MaxAttempts
	if max <= 0 {
		max = 1
	}

	var lastErr error
	for attempt := 0; attempt < max; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		val, err := fn()
		if err == nil {
			return val, nil
		}
		lastErr = err
		if !IsTransient(err) || attempt+1 >= max {
			return zero, err
		}
		wait := Backoff(p, attempt, RetryAfter(err))
		if onRetry != nil {
			onRetry(attempt+1, max, wait, err)
		}
		if wait <= 0 {
			continue
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("retry: exhausted %d attempts", max)
	}
	return zero, lastErr
}
