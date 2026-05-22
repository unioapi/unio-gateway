package ledger

import (
	"context"

	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5"
)

// TxBeginner 定义 ledger service 开启数据库事务所需能力。
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Service 负责用户余额变动和账本流水写入。
type Service struct {
	db      TxBeginner
	queries *sqlc.Queries
}

// TODO(阶段7/production): [GAP-7-011] ledger reservation 已具备 pre-authorize/capture/release，但 gateway 仍未在调用上游前冻结余额，失败/取消/无 final usage 路径也未统一 release；公开计费 API 前；接入 gateway preauthorization 并收口异常释放策略。

// NewService 创建 ledger service。
func NewService(db TxBeginner, queries *sqlc.Queries) *Service {
	return &Service{db: db, queries: queries}
}
