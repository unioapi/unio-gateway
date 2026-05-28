package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ThankCat/unio-api/internal/core/auth"
	"github.com/ThankCat/unio-api/internal/platform/failure"
	"github.com/ThankCat/unio-api/internal/platform/httpx"
	"github.com/ThankCat/unio-api/internal/platform/ratelimit"
)

// RateLimitFailurePolicy 表示限流器故障时的处理策略。
type RateLimitFailurePolicy string

const (
	// RateLimitFailurePolicyFailClosed 表示限流器故障时拒绝请求。
	RateLimitFailurePolicyFailClosed RateLimitFailurePolicy = "fail_closed"

	// RateLimitFailurePolicyFailOpen 表示限流器故障时放行请求。
	RateLimitFailurePolicyFailOpen RateLimitFailurePolicy = "fail_open"
)

const (
	HeaderRateLimitLimit         = "X-RateLimit-Limit"
	HeaderRateLimitRemaining     = "X-RateLimit-Remaining"
	HeaderRateLimitReset         = "X-RateLimit-Reset"
	rateLimitSubjectAPIKeyPrefix = "api_key:"
)

// RateLimitOptions 保存 RateLimit middleware 的运行参数。
type RateLimitOptions struct {
	Limit         int64
	Window        time.Duration
	FailurePolicy RateLimitFailurePolicy
	Logger        *slog.Logger
}

// RateLimiter 定义 middleware 调用限流器所需的最小能力。
type RateLimiter interface {
	Allow(ctx context.Context, subject string, limit int64, window time.Duration) (ratelimit.Decision, error)
}

// RateLimit 使用认证身份作为 subject，对请求做基础限流。
func RateLimit(limiter RateLimiter, opts RateLimitOptions) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := auth.APIKeyPrincipalFromContext(r.Context())
			if !ok {
				_ = httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "missing api key principal")
				return
			}

			subject := apiKeyRateLimitSubject(principal.APIKeyID)
			decision, err := limiter.Allow(r.Context(), subject, opts.Limit, opts.Window)
			if err != nil {
				logRateLimitFailure(opts.Logger, subject, principal.KeyPrefix, opts.FailurePolicy, err)

				if opts.FailurePolicy == RateLimitFailurePolicyFailOpen {
					next.ServeHTTP(w, r)
					return
				}

				_ = httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "rate limit failed")
				return
			}

			writeRateLimitHeaders(w, decision)

			if !decision.Allowed {
				_ = httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// writeRateLimitHeaders 写入标准限流响应头。
func writeRateLimitHeaders(w http.ResponseWriter, decision ratelimit.Decision) {
	w.Header().Set(HeaderRateLimitLimit, strconv.FormatInt(decision.Limit, 10))
	w.Header().Set(HeaderRateLimitRemaining, strconv.FormatInt(decision.Remaining, 10))
	w.Header().Set(HeaderRateLimitReset, strconv.FormatInt(decision.ResetAt.Unix(), 10))
}

// apiKeyRateLimitSubject 返回 API Key 对应的限流 subject。
func apiKeyRateLimitSubject(apiKeyID int64) string {
	return rateLimitSubjectAPIKeyPrefix + strconv.FormatInt(apiKeyID, 10)
}

// logRateLimitFailure 记录限流器故障；只记录 key prefix，不记录完整 API key。
func logRateLimitFailure(logger *slog.Logger, subject string, keyPrefix string, policy RateLimitFailurePolicy, err error) {
	if logger == nil {
		return
	}

	args := []any{
		"subject", subject,
		"api_key_prefix", keyPrefix,
		"failure_policy", string(policy),
	}
	args = append(args, failure.LogArgs(err)...)

	logger.Warn("rate limit failed", args...)
}
