package apikey

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ThankCat/unio-api/internal/failure"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	// ErrInvalidProjectID 表示创建 API Key 时 project_id 非法。
	ErrInvalidProjectID = errors.New("invalid project id")

	// ErrInvalidName 表示创建 API Key 时 name 为空。
	ErrInvalidName = errors.New("invalid api key name")

	// ErrUnauthorizedProject 表示调用者无权操作目标 project。
	ErrUnauthorizedProject = errors.New("unauthorized project")
)

// Store 定义 API Key 创建服务需要的数据库能力。
type Store interface {
	CreateAPIKey(ctx context.Context, arg sqlc.CreateAPIKeyParams) (sqlc.ApiKey, error)
	GetProjectForUser(ctx context.Context, arg sqlc.GetProjectForUserParams) (sqlc.Project, error)
}

// Service 负责 API Key 的业务创建流程。
type Service struct {
	store Store
}

// NewService 创建 API Key service。
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreatedKey 表示创建成功后返回给调用方的一次性结果。
type CreatedKey struct {
	ID        int64
	ProjectID int64
	Name      string
	Plaintext string
	Prefix    string
	ExpiresAt *time.Time
}

// CreateParams 表示创建 API Key 需要的业务参数。
type CreateParams struct {
	ProjectID   int64
	Name        string
	ExpiresAt   *time.Time
	ActorUserID int64
}

// Create 创建新的 API Key。Plaintext 只能返回给调用方一次，不能保存到数据库。
func (s *Service) Create(ctx context.Context, params CreateParams) (*CreatedKey, error) {
	// TODO(阶段3/production): [GAP-3-007] API Key 创建缺少审计日志；开放 key 管理 API 前；接入 audit log 记录 actor、project、api_key 和操作结果。
	if params.ProjectID <= 0 {
		return nil, failure.Wrap(
			failure.CodeAPIKeyInvalidProjectID,
			ErrInvalidProjectID,
			failure.WithMessage(ErrInvalidProjectID.Error()),
		)
	}

	if params.ActorUserID <= 0 {
		return nil, failure.Wrap(
			failure.CodeAPIKeyUnauthorizedProject,
			ErrUnauthorizedProject,
			failure.WithMessage(ErrUnauthorizedProject.Error()),
		)
	}

	if _, err := s.store.GetProjectForUser(ctx, sqlc.GetProjectForUserParams{
		ProjectID: params.ProjectID,
		UserID:    params.ActorUserID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, failure.Wrap(
				failure.CodeAPIKeyUnauthorizedProject,
				ErrUnauthorizedProject,
				failure.WithMessage(ErrUnauthorizedProject.Error()),
			)
		}

		return nil, failure.Wrap(
			failure.CodeAPIKeyStoreFailed,
			err,
			failure.WithMessage("lookup project for api key"),
		)
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, failure.Wrap(
			failure.CodeAPIKeyInvalidName,
			ErrInvalidName,
			failure.WithMessage(ErrInvalidName.Error()),
		)
	}

	generatedKey, err := Generate()
	if err != nil {
		return nil, failure.Wrap(
			failure.CodeAPIKeyGenerateFailed,
			err,
			failure.WithMessage("generate api key"),
		)
	}

	expiresAt := pgtype.Timestamptz{Valid: false}
	if params.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *params.ExpiresAt, Valid: true}
	}

	storeParams := sqlc.CreateAPIKeyParams{
		ProjectID: params.ProjectID,
		Name:      name,
		KeyPrefix: generatedKey.Prefix,
		KeyHash:   generatedKey.Hash,
		ExpiresAt: expiresAt,
	}
	storedKey, err := s.store.CreateAPIKey(ctx, storeParams)
	if err != nil {
		return nil, failure.Wrap(
			failure.CodeAPIKeyStoreFailed,
			err,
			failure.WithMessage("create api key"),
		)
	}

	var createdExpiresAt *time.Time
	if storedKey.ExpiresAt.Valid {
		t := storedKey.ExpiresAt.Time
		createdExpiresAt = &t
	}

	return &CreatedKey{
		ID:        storedKey.ID,
		ProjectID: storedKey.ProjectID,
		Name:      storedKey.Name,
		Plaintext: generatedKey.Plaintext,
		Prefix:    storedKey.KeyPrefix,
		ExpiresAt: createdExpiresAt,
	}, nil
}
