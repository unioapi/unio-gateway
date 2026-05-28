package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeStore 是限流器测试使用的计数存储替身。
type fakeStore struct {
	called bool
	key    string
	window time.Duration
	result CountResult
	err    error
}

// Increment 记录限流器传入的 key 和 window，并返回测试预设的计数结果。
func (f *fakeStore) Increment(ctx context.Context, key string, window time.Duration) (CountResult, error) {
	f.called = true
	f.key = key
	f.window = window
	return f.result, f.err
}

func TestLimiterAllowWithinLimit(t *testing.T) {
	resetAt := time.Date(2026, 5, 8, 10, 1, 0, 0, time.UTC)
	store := &fakeStore{
		result: CountResult{
			Count:   3,
			ResetAt: resetAt,
		},
	}
	limiter := NewLimiter(store)

	decision, err := limiter.Allow(context.Background(), "api_key:1", 5, time.Minute)
	if err != nil {
		t.Fatalf("allow: %v", err)
	}

	if !store.called {
		t.Fatal("want store to be called")
	}

	if store.key != "api_key:1" {
		t.Fatalf("want store key %q, got %q", "api_key:1", store.key)
	}

	if store.window != time.Minute {
		t.Fatalf("want store window %v, got %v", time.Minute, store.window)
	}

	if !decision.Allowed {
		t.Fatal("want request to be allowed")
	}

	if decision.Limit != 5 {
		t.Fatalf("want limit 5, got %d", decision.Limit)
	}

	if decision.Remaining != 2 {
		t.Fatalf("want remaining 2, got %d", decision.Remaining)
	}

	if !decision.ResetAt.Equal(resetAt) {
		t.Fatalf("want reset_at %v, got %v", resetAt, decision.ResetAt)
	}
}

func TestLimiterRejectOverLimit(t *testing.T) {
	resetAt := time.Date(2026, 5, 8, 10, 1, 0, 0, time.UTC)
	store := &fakeStore{
		result: CountResult{
			Count:   6,
			ResetAt: resetAt,
		},
	}
	limiter := NewLimiter(store)

	decision, err := limiter.Allow(context.Background(), "api_key:1", 5, time.Minute)
	if err != nil {
		t.Fatalf("allow: %v", err)
	}

	if !store.called {
		t.Fatal("want store to be called")
	}

	if decision.Allowed {
		t.Fatal("want request to be rejected")
	}

	if decision.Limit != 5 {
		t.Fatalf("want limit 5, got %d", decision.Limit)
	}

	if decision.Remaining != 0 {
		t.Fatalf("want remaining 0, got %d", decision.Remaining)
	}

	if !decision.ResetAt.Equal(resetAt) {
		t.Fatalf("want reset_at %v, got %v", resetAt, decision.ResetAt)
	}
}

func TestLimiterInvalidSubject(t *testing.T) {
	store := &fakeStore{}
	limiter := NewLimiter(store)

	_, err := limiter.Allow(context.Background(), "   ", 5, time.Minute)
	if !errors.Is(err, ErrInvalidSubject) {
		t.Fatalf("want ErrInvalidSubject, got %v", err)
	}

	if store.called {
		t.Fatal("want store not to be called")
	}
}

func TestLimiterInvalidLimit(t *testing.T) {
	store := &fakeStore{}
	limiter := NewLimiter(store)

	_, err := limiter.Allow(context.Background(), "api_key:1", 0, time.Minute)
	if !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("want ErrInvalidLimit, got %v", err)
	}

	if store.called {
		t.Fatal("want store not to be called")
	}
}

func TestLimiterInvalidWindow(t *testing.T) {
	store := &fakeStore{}
	limiter := NewLimiter(store)

	_, err := limiter.Allow(context.Background(), "api_key:1", 5, 0)
	if !errors.Is(err, ErrInvalidWindow) {
		t.Fatalf("want ErrInvalidWindow, got %v", err)
	}

	if store.called {
		t.Fatal("want store not to be called")
	}
}

func TestLimiterStoreError(t *testing.T) {
	storeErr := errors.New("increment failed")
	store := &fakeStore{err: storeErr}
	limiter := NewLimiter(store)

	_, err := limiter.Allow(context.Background(), "api_key:1", 5, time.Minute)
	if !errors.Is(err, storeErr) {
		t.Fatalf("want store error, got %v", err)
	}

	if !store.called {
		t.Fatal("want store to be called")
	}
}
