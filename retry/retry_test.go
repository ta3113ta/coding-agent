package retry

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestIsTransient(t *testing.T) {
	t.Parallel()
	if IsTransient(nil) {
		t.Fatal("nil should not be transient")
	}
	if IsTransient(errors.New("permanent")) {
		t.Fatal("plain error should not be transient")
	}
	if !IsTransient(Transient(errors.New("boom"))) {
		t.Fatal("Transient-wrapped should be transient")
	}
	if IsTransient(context.Canceled) {
		t.Fatal("context.Canceled must not be transient")
	}
	if IsTransient(Transient(context.Canceled)) {
		// Still wrapped, but errors.Is checks cancel first — wait, our IsTransient
		// checks Canceled before Transient. Transient(context.Canceled) still
		// errors.Is to Canceled, so should be false.
		t.Fatal("canceled must not be transient even if wrapped")
	}
}

func TestRetryAfter(t *testing.T) {
	t.Parallel()
	err := Transient(errors.New("rate"), 5*time.Second)
	if got := RetryAfter(err); got != 5*time.Second {
		t.Fatalf("RetryAfter = %v, want 5s", got)
	}
	if RetryAfter(errors.New("x")) != 0 {
		t.Fatal("expected 0")
	}
}

func TestParseRetryAfterHeader(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("Retry-After", "7")
	if got := ParseRetryAfterHeader(h); got != 7*time.Second {
		t.Fatalf("got %v", got)
	}
}

func TestHTTPStatusTransient(t *testing.T) {
	t.Parallel()
	for _, code := range []int{408, 409, 429, 500, 503, 529} {
		if !HTTPStatusTransient(code) {
			t.Fatalf("%d should be transient", code)
		}
	}
	for _, code := range []int{400, 401, 403, 404} {
		if HTTPStatusTransient(code) {
			t.Fatalf("%d should not be transient", code)
		}
	}
}

func TestDoMaxAttemptsOne(t *testing.T) {
	t.Parallel()
	calls := 0
	_, err := Do(context.Background(), Policy{MaxAttempts: 1}, func() (int, error) {
		calls++
		return 0, Transient(errors.New("fail"))
	}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestDoRetriesTransient(t *testing.T) {
	t.Parallel()
	calls := 0
	val, err := Do(context.Background(), Policy{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
	}, func() (int, error) {
		calls++
		if calls < 3 {
			return 0, Transient(errors.New("temp"))
		}
		return 42, nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if val != 42 || calls != 3 {
		t.Fatalf("val=%d calls=%d", val, calls)
	}
}

func TestDoPermanentNoRetry(t *testing.T) {
	t.Parallel()
	calls := 0
	_, err := Do(context.Background(), Policy{
		MaxAttempts:    5,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	}, func() (int, error) {
		calls++
		return 0, errors.New("auth failed")
	}, nil)
	if err == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestDoContextCancelDuringWait(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	_, err := Do(ctx, Policy{
		MaxAttempts:    5,
		InitialBackoff: 2 * time.Second,
		MaxBackoff:     2 * time.Second,
	}, func() (int, error) {
		calls++
		if calls == 1 {
			cancel()
			return 0, Transient(errors.New("temp"))
		}
		return 1, nil
	}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want canceled, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d", calls)
	}
}

func TestBackoffRespectsRetryAfter(t *testing.T) {
	t.Parallel()
	p := Policy{InitialBackoff: time.Millisecond, MaxBackoff: time.Second}
	got := Backoff(p, 0, 500*time.Millisecond)
	// With jitter, wait is in [500ms, 600ms]
	if got < 500*time.Millisecond || got > 600*time.Millisecond {
		t.Fatalf("got %v", got)
	}
}
