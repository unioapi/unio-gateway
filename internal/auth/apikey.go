package auth

import (
	"context"
	"errors"
	"time"

	"github.com/ThankCat/unio-api/internal/apikey"
	"github.com/ThankCat/unio-api/internal/failure"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	// ErrMissingAPIKey 表示请求没有提供 API Key。
	ErrMissingAPIKey = errors.New("missing api key")
	// ErrInvalidAPIKey 表示 API Key 不存在或无法匹配。
	ErrInvalidAPIKey = errors.New("invalid api key")
	// ErrAPIKeyRevoked 表示 API Key 已被永久吊销。
	ErrAPIKeyRevoked = errors.New("api key revoked")
	// ErrAPIKeyDisabled 表示 API Key 被临时禁用。
	ErrAPIKeyDisabled = errors.New("api key disabled")
	// ErrAPIKeyExpired 表示 API Key 已经过期。
	ErrAPIKeyExpired = errors.New("api key expired")
)

// APIKeyPrincipal 表示 API Key 认证成功后的请求身份。
type APIKeyPrincipal struct {
	APIKeyID  int64
	UserID    int64
	ProjectID int64
	KeyPrefix string
}

// APIKeyStore 定义 API Key 认证所需的存储查询和更新能力。
type APIKeyStore interface {
	GetAPIKeyByHash(ctx context.Context, keyHash string) (sqlc.GetAPIKeyByHashRow, error)
	UpdateAPIKeyLastUsedAt(ctx context.Context, arg sqlc.UpdateAPIKeyLastUsedAtParams) error
}

// APIKeyAuthenticator 负责校验 API Key 并生成认证身份。
type APIKeyAuthenticator struct {
	store APIKeyStore
	now   func() time.Time
}

// NewAPIKeyAuthenticator 创建 APIKeyAuthenticator。
func NewAPIKeyAuthenticator(store APIKeyStore) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		store: store,
		now:   time.Now,
	}
}

// AuthenticateAPIKey 校验明文 API Key，并返回认证后的请求身份。
func (a *APIKeyAuthenticator) AuthenticateAPIKey(ctx context.Context, plaintext string) (*APIKeyPrincipal, error) {
	if plaintext == "" {
		return nil, failure.Wrap(
			failure.CodeAuthMissingAPIKey,
			ErrMissingAPIKey,
			failure.WithMessage(ErrMissingAPIKey.Error()),
		)
	}

	keyHash := apikey.Hash(plaintext)

	key, err := a.store.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, failure.Wrap(
				failure.CodeAuthInvalidAPIKey,
				ErrInvalidAPIKey,
				failure.WithMessage(ErrInvalidAPIKey.Error()),
			)
		}
		return nil, failure.Wrap(
			failure.CodeAuthStoreFailed,
			err,
			failure.WithMessage("lookup api key"),
		)
	}

	if key.RevokedAt.Valid {
		return nil, failure.Wrap(
			failure.CodeAuthAPIKeyRevoked,
			ErrAPIKeyRevoked,
			failure.WithMessage(ErrAPIKeyRevoked.Error()),
		)
	}

	if key.DisabledAt.Valid {
		return nil, failure.Wrap(
			failure.CodeAuthAPIKeyDisabled,
			ErrAPIKeyDisabled,
			failure.WithMessage(ErrAPIKeyDisabled.Error()),
		)
	}

	if key.ExpiresAt.Valid && !key.ExpiresAt.Time.After(a.now()) {
		return nil, failure.Wrap(
			failure.CodeAuthAPIKeyExpired,
			ErrAPIKeyExpired,
			failure.WithMessage(ErrAPIKeyExpired.Error()),
		)
	}

	// TODO(阶段3/production): [GAP-3-001] 每次认证同步更新 last_used_at 会放大数据库写入；后续评估节流、异步或批量更新策略。
	// 更新最后使用时间
	usedAt := a.now()
	if err := a.store.UpdateAPIKeyLastUsedAt(ctx, sqlc.UpdateAPIKeyLastUsedAtParams{
		LastUsedAt: pgtype.Timestamptz{Time: usedAt, Valid: true},
		ID:         key.ID,
	}); err != nil {
		return nil, failure.Wrap(
			failure.CodeAuthStoreFailed,
			err,
			failure.WithMessage("update api key last used at"),
		)
	}

	return &APIKeyPrincipal{
		APIKeyID:  key.ID,
		UserID:    key.UserID,
		ProjectID: key.ProjectID,
		KeyPrefix: key.KeyPrefix,
	}, nil
}

// TODO(阶段3/production): [GAP-3-002] 补齐 API Key revoke、disable、list 和审计日志能力，确保后台能安全管理 customer API key。
