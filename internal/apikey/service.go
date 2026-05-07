package apikey

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	// ErrInvalidProjectID 表示创建 API Key 时 project_id 非法。
	ErrInvalidProjectID = errors.New("invalid project id")

	// ErrInvalidName 表示创建 API Key 时 name 为空。
	ErrInvalidName = errors.New("invalid api key name")
)

// Store 定义 API Key 创建服务需要的数据库能力。
type Store interface {
	CreateAPIKey(ctx context.Context, arg sqlc.CreateAPIKeyParams) (sqlc.ApiKey, error)
}

// Service 负责 API Key 的业务创建流程。
type Service struct {
	store Store
}

// Create 创建新的 API Key。Plaintext 只能返回给调用方一次，不能保存到数据库。
func (s *Service) Create(ctx context.Context, params CreateParams) (*CreatedKey, error) {
	if params.ProjectID <= 0 {
		return nil, ErrInvalidProjectID
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, ErrInvalidName
	}

	generatedKey, err := Generate()
	if err != nil {
		return nil, err
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
	apiKey, err := s.store.CreateAPIKey(ctx, storeParams)
	if err != nil {
		return nil, err
	}

	var createdExpiresAt *time.Time
	if apiKey.ExpiresAt.Valid {
		t := apiKey.ExpiresAt.Time
		createdExpiresAt = &t
	}

	return &CreatedKey{
		ID:        apiKey.ID,
		ProjectID: apiKey.ProjectID,
		Name:      apiKey.Name,
		Plaintext: generatedKey.Plaintext,
		Prefix:    apiKey.KeyPrefix,
		ExpiresAt: createdExpiresAt,
	}, nil
}

// CreateParams 表示创建 API Key 需要的业务参数。
type CreateParams struct {
	ProjectID int64
	Name      string
	ExpiresAt *time.Time
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

// NewService 创建 API Key service。
func NewService(store Store) *Service {
	return &Service{store: store}
}
