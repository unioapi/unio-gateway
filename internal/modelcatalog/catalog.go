package modelcatalog

import (
	"context"

	"github.com/ThankCat/unio-api/internal/store/sqlc"
)

// Model 表示 Unio 对外可见的模型。
type Model struct {
	ID      string
	OwnedBy string
}

// Store 定义 model catalog 读取可用模型所需的最小数据库能力。
type Store interface {
	ListAvailableModelsForProject(ctx context.Context, projectID int64) ([]sqlc.ListAvailableModelsForProjectRow, error)
}

// Service 负责查询当前 project 可见的模型列表。
type Service struct {
	store Store
}

// NewService 创建 model catalog service。
func NewService(store Store) *Service {
	return &Service{store: store}
}

// ListAvailableModels 返回当前 project 可见的 OpenAI-compatible 模型。
func (s *Service) ListAvailableModels(ctx context.Context, projectID int64) ([]Model, error) {
	// TODO(阶段6/production): [GAP-6-006] /v1/models 当前只按全局 enabled channel/model 返回，未体现 project 级可见性、预算或禁用策略；开放后台项目配置前；与 routing 共用 project model/channel policy，保证“可见模型”和“可路由模型”一致。
	rows, err := s.store.ListAvailableModelsForProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	models := make([]Model, 0, len(rows))
	for _, row := range rows {
		models = append(models, Model{
			ID:      row.ModelID,
			OwnedBy: row.OwnedBy,
		})
	}

	return models, nil
}
