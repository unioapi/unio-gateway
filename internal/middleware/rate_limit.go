package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/httpx"
	"github.com/ThankCat/unio-api/internal/ratelimit"
)

const (
	HeaderRateLimitLimit         = "X-RateLimit-Limit"
	HeaderRateLimitRemaining     = "X-RateLimit-Remaining"
	HeaderRateLimitReset         = "X-RateLimit-Reset"
	rateLimitSubjectAPIKeyPrefix = "api_key:"
)

// RateLimiter 定义 middleware 调用限流器所需的最小能力。
type RateLimiter interface {
	Allow(ctx context.Context, subject string, limit int64, window time.Duration) (ratelimit.Decision, error)
}

// RateLimit 使用认证身份作为 subject，对请求做基础限流。
func RateLimit(limiter RateLimiter, limit int64, window time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := auth.APIKeyPrincipalFromContext(r.Context())
			if !ok {
				_ = httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "missing api key principal")
				return
			}

			subject := apiKeyRateLimitSubject(principal.APIKeyID)
			decision, err := limiter.Allow(r.Context(), subject, limit, window)
			if err != nil {
				// TODO(阶段3/production): [GAP-3-006] Redis 限流故障当前会让客户请求全部失败，可能形成单点不可用；生产部署前；将 fail-open/fail-closed 策略配置化，并补充降级日志/metrics。
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
