package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/httpx"
	"github.com/ThankCat/unio-api/internal/ratelimit"
)

// fakeRateLimiter 是 RateLimit middleware 测试使用的限流器替身。
type fakeRateLimiter struct {
	subject  string
	limit    int64
	window   time.Duration
	decision ratelimit.Decision
	err      error
}

// Allow 记录 middleware 传入的限流参数，并返回测试预设的判断结果。
func (l *fakeRateLimiter) Allow(ctx context.Context, subject string, limit int64, window time.Duration) (ratelimit.Decision, error) {
	l.subject = subject
	l.limit = limit
	l.window = window
	return l.decision, l.err
}

func TestRateLimitAllowsRequest(t *testing.T) {
	resetAt := time.Date(2026, 5, 8, 10, 1, 0, 0, time.UTC)
	limiter := &fakeRateLimiter{
		decision: ratelimit.Decision{
			Allowed:   true,
			Limit:     60,
			Remaining: 59,
			ResetAt:   resetAt,
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := RateLimit(limiter, 60, time.Minute)(next)
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req = req.WithContext(auth.ContextWithAPIKeyPrincipal(req.Context(), &auth.APIKeyPrincipal{
		APIKeyID:  123,
		ProjectID: 456,
		KeyPrefix: "unio_sk_test",
	}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !nextCalled {
		t.Fatal("want next handler to be called")
	}

	if rec.Code != http.StatusNoContent {
		t.Fatalf("want status %d, got %d", http.StatusNoContent, rec.Code)
	}

	if limiter.subject != "api_key:123" {
		t.Fatalf("want subject %q, got %q", "api_key:123", limiter.subject)
	}

	if limiter.limit != 60 {
		t.Fatalf("want limit 60, got %d", limiter.limit)
	}

	if limiter.window != time.Minute {
		t.Fatalf("want window %v, got %v", time.Minute, limiter.window)
	}

	assertRateLimitHeaders(t, rec, "60", "59", resetAt)
}

func TestRateLimitRejectsRequest(t *testing.T) {
	resetAt := time.Date(2026, 5, 8, 10, 1, 0, 0, time.UTC)
	limiter := &fakeRateLimiter{
		decision: ratelimit.Decision{
			Allowed:   false,
			Limit:     60,
			Remaining: 0,
			ResetAt:   resetAt,
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	handler := RateLimit(limiter, 60, time.Minute)(next)
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req = req.WithContext(auth.ContextWithAPIKeyPrincipal(req.Context(), &auth.APIKeyPrincipal{
		APIKeyID:  123,
		ProjectID: 456,
		KeyPrefix: "unio_sk_test",
	}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if nextCalled {
		t.Fatal("want next handler not to be called")
	}

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("want status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}

	assertRateLimitHeaders(t, rec, "60", "0", resetAt)

	var body httpx.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if body.Error.Code != "rate_limited" {
		t.Fatalf("want error code %q, got %q", "rate_limited", body.Error.Code)
	}
}

func TestRateLimitMissingPrincipal(t *testing.T) {
	limiter := &fakeRateLimiter{}
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	handler := RateLimit(limiter, 60, time.Minute)(next)
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if nextCalled {
		t.Fatal("want next handler not to be called")
	}

	if limiter.subject != "" {
		t.Fatalf("want limiter not to be called, got subject %q", limiter.subject)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestRateLimitLimiterError(t *testing.T) {
	limiter := &fakeRateLimiter{
		err: errors.New("rate limit failed"),
	}
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	handler := RateLimit(limiter, 60, time.Minute)(next)
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req = req.WithContext(auth.ContextWithAPIKeyPrincipal(req.Context(), &auth.APIKeyPrincipal{
		APIKeyID:  123,
		ProjectID: 456,
		KeyPrefix: "unio_sk_test",
	}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if nextCalled {
		t.Fatal("want next handler not to be called")
	}

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	if limiter.subject != "api_key:123" {
		t.Fatalf("want subject %q, got %q", "api_key:123", limiter.subject)
	}
}

func TestAPIKeyRateLimitSubject(t *testing.T) {
	got := apiKeyRateLimitSubject(123)
	want := "api_key:123"
	if got != want {
		t.Fatalf("want subject %q, got %q", want, got)
	}
}

// assertRateLimitHeaders 校验响应中的限流 header。
func assertRateLimitHeaders(t *testing.T, rec *httptest.ResponseRecorder, limit string, remaining string, resetAt time.Time) {
	t.Helper()

	if rec.Header().Get(HeaderRateLimitLimit) != limit {
		t.Fatalf("want %s %q, got %q", HeaderRateLimitLimit, limit, rec.Header().Get(HeaderRateLimitLimit))
	}

	if rec.Header().Get(HeaderRateLimitRemaining) != remaining {
		t.Fatalf("want %s %q, got %q", HeaderRateLimitRemaining, remaining, rec.Header().Get(HeaderRateLimitRemaining))
	}

	wantReset := strconvFormatUnix(resetAt)
	if rec.Header().Get(HeaderRateLimitReset) != wantReset {
		t.Fatalf("want %s %q, got %q", HeaderRateLimitReset, wantReset, rec.Header().Get(HeaderRateLimitReset))
	}
}

// strconvFormatUnix 返回时间的 Unix 秒字符串，避免测试里重复格式化逻辑。
func strconvFormatUnix(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}
