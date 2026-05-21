package ratelimit

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ThankCat/unio-api/internal/failure"
)

var (
	// ErrInvalidSubject 表示限流 subject 为空。
	ErrInvalidSubject = errors.New("invalid rate limit subject")

	// ErrInvalidLimit 表示限流次数配置非法。
	ErrInvalidLimit = errors.New("invalid rate limit limit")

	// ErrInvalidWindow 表示限流窗口配置非法。
	ErrInvalidWindow = errors.New("invalid rate limit window")
)

// Store 定义限流器需要的计数存储能力。
type Store interface {
	Increment(ctx context.Context, key string, window time.Duration) (CountResult, error)
}

// CountResult 表示一次计数后的结果。
type CountResult struct {
	Count   int64
	ResetAt time.Time
}

// Limiter 负责根据计数结果判断请求是否允许通过。
type Limiter struct {
	store Store
}

// Decision 表示一次限流判断结果。
type Decision struct {
	Allowed   bool
	Limit     int64
	Remaining int64
	ResetAt   time.Time
}

// NewLimiter 创建限流器。
func NewLimiter(store Store) *Limiter {
	return &Limiter{store: store}
}

// Allow 判断 subject 在指定窗口内是否还能继续请求。
func (l *Limiter) Allow(ctx context.Context, subject string, limit int64, window time.Duration) (Decision, error) {
	if strings.TrimSpace(subject) == "" {
		return Decision{}, failure.Wrap(
			failure.CodeRateLimitInvalidSubject,
			ErrInvalidSubject,
			failure.WithMessage(ErrInvalidSubject.Error()),
		)
	}

	if limit <= 0 {
		return Decision{}, failure.Wrap(
			failure.CodeRateLimitInvalidLimit,
			ErrInvalidLimit,
			failure.WithMessage(ErrInvalidLimit.Error()),
		)
	}

	if window <= 0 {
		return Decision{}, failure.Wrap(
			failure.CodeRateLimitInvalidWindow,
			ErrInvalidWindow,
			failure.WithMessage(ErrInvalidWindow.Error()),
		)
	}

	result, err := l.store.Increment(ctx, subject, window)
	if err != nil {
		return Decision{}, failure.Wrap(
			failure.CodeRateLimitStoreFailed,
			err,
			failure.WithMessage("increment rate limit counter"),
		)
	}

	if result.Count <= limit {
		return Decision{
			Allowed:   true,
			Limit:     limit,
			Remaining: limit - result.Count,
			ResetAt:   result.ResetAt,
		}, nil
	}

	return Decision{
		Allowed:   false,
		Limit:     limit,
		Remaining: 0,
		ResetAt:   result.ResetAt,
	}, nil
}
