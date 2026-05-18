package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/billing"
	"github.com/ThankCat/unio-api/internal/ledger"
	"github.com/ThankCat/unio-api/internal/requestlog"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ChatTxBeginner 定义 chat settlement 开启数据库事务所需能力。
type ChatTxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// ChatLedgerDebiter 定义 chat settlement 扣减用户余额所需能力。
type ChatLedgerDebiter interface {
	DebitWithQueries(ctx context.Context, queries *sqlc.Queries, params ledger.DebitParams) (ledger.Entry, error)
}

// ChatBillingCalculator 定义 chat settlement 计算请求金额所需能力。
type ChatBillingCalculator interface {
	Calculate(usage billing.Usage, price billing.PriceSnapshot) (billing.Settlement, error)
}

// ChatSettlementService 负责非流式 chat 请求成功后的 usage、price snapshot 和 ledger 结算。
type ChatSettlementService struct {
	db                ChatTxBeginner
	queries           *sqlc.Queries
	billingCalculator ChatBillingCalculator
	ledgerDebiter     ChatLedgerDebiter
}

// NewChatSettlementService 创建 chat 请求结算 service。
func NewChatSettlementService(db ChatTxBeginner, queries *sqlc.Queries, billingCalculator ChatBillingCalculator, ledgerDebiter ChatLedgerDebiter) *ChatSettlementService {
	if db == nil {
		panic("gateway: chat settlement tx beginner is required")
	}
	if queries == nil {
		panic("gateway: chat settlement queries is required")
	}
	if billingCalculator == nil {
		panic("gateway: chat billing calculator is required")
	}
	if ledgerDebiter == nil {
		panic("gateway: chat ledger debiter is required")
	}

	return &ChatSettlementService{
		db:                db,
		queries:           queries,
		billingCalculator: billingCalculator,
		ledgerDebiter:     ledgerDebiter,
	}
}

// ChatSettlementParams 表示一次成功 chat 请求结算所需的事实。
type ChatSettlementParams struct {
	RequestRecord   requestlog.RequestRecord
	AttemptRecord   requestlog.AttemptRecord
	Principal       *auth.APIKeyPrincipal
	ResponseModelID string
	ModelDBID       int64
	FinalProviderID int64
	FinalChannelID  int64
	AdapterResp     *adapter.ChatResponse
}

// SettleSuccessfulChat 对一次成功的非流式 chat 请求执行结算。
func (s *ChatSettlementService) SettleSuccessfulChat(ctx context.Context, params ChatSettlementParams) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	now := time.Now()
	txQueries := s.queries.WithTx(tx)
	txRequestLog := requestlog.NewStore(txQueries)
	usage := params.AdapterResp.Usage

	// TODO(阶段8/production): adapter 成功响应缺少真实 upstream status/request id 会影响渠道审计精度；接入 provider error classification / adapter metadata 时；从 adapter response metadata 写入真实 upstream_status_code 和 upstream_request_id。
	_, err = txRequestLog.MarkAttemptSucceeded(ctx, requestlog.MarkAttemptSucceededParams{
		ID:                    params.AttemptRecord.ID,
		UpstreamResponseModel: params.AdapterResp.Model,
		UpstreamStatusCode:    200,
		UpstreamRequestID:     nil,
		CompletedAt:           now,
	})
	if err != nil {
		return err
	}

	// TODO(阶段7/production): adapter.ChatUsage 暂未携带 cached/reasoning 明细会低估或误分摊特殊 token 费用；接入 provider usage 明细映射前；扩展 adapter.ChatUsage 并由各 adapter 映射上游 cached/reasoning usage。
	_, err = txQueries.CreateUsageRecord(ctx, sqlc.CreateUsageRecordParams{
		RequestRecordID:  params.RequestRecord.ID,
		PromptTokens:     int64(usage.PromptTokens),
		CompletionTokens: int64(usage.CompletionTokens),
		TotalTokens:      int64(usage.TotalTokens),
		CachedTokens:     0,
		ReasoningTokens:  0,
		Source:           "upstream_response",
	})
	if err != nil {
		return err
	}

	// 按 routing 选中的模型数据库主键查询当前生效售卖价；不能用对外 model_id 字符串直接查价格。
	price, err := txQueries.FindActivePriceForModel(ctx, sqlc.FindActivePriceForModelParams{
		ModelID: params.ModelDBID,
		AtTime:  pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return err
	}

	// 将当前价格复制成请求级快照，保证后续价格调整不会改变历史请求的结算依据。
	snapshot, err := txQueries.CreatePriceSnapshot(ctx, sqlc.CreatePriceSnapshotParams{
		RequestRecordID:      params.RequestRecord.ID,
		PriceID:              pgtype.Int8{Int64: price.ID, Valid: true},
		Currency:             price.Currency,
		PricingUnit:          price.PricingUnit,
		InputPrice:           price.InputPrice,
		OutputPrice:          price.OutputPrice,
		CachedInputPrice:     price.CachedInputPrice,
		ReasoningOutputPrice: price.ReasoningOutputPrice,
		FormulaVersion:       billing.FormulaVersionV1,
	})
	if err != nil {
		return err
	}

	// billing 只做纯金额计算；它不写 usage、不查价格、不扣余额。
	settlement, err := s.billingCalculator.Calculate(billing.Usage{
		PromptTokens:     int64(usage.PromptTokens),
		CompletionTokens: int64(usage.CompletionTokens),
		TotalTokens:      int64(usage.TotalTokens),
		CachedTokens:     0,
		ReasoningTokens:  0,
	}, billing.PriceSnapshot{
		Currency:             snapshot.Currency,
		PricingUnit:          snapshot.PricingUnit,
		InputPrice:           snapshot.InputPrice,
		OutputPrice:          snapshot.OutputPrice,
		CachedInputPrice:     snapshot.CachedInputPrice,
		ReasoningOutputPrice: snapshot.ReasoningOutputPrice,
		FormulaVersion:       snapshot.FormulaVersion,
	})
	if err != nil {
		return err
	}

	// ledger_entries.amount 要求大于 0；零金额请求保留 usage 和 price snapshot，但不写余额流水。
	if !numericIsZero(settlement.Amount) {
		_, err = s.ledgerDebiter.DebitWithQueries(ctx, txQueries, ledger.DebitParams{
			UserID:          params.Principal.UserID,
			RequestRecordID: &params.RequestRecord.ID,
			Amount:          settlement.Amount,
			Currency:        settlement.Currency,
			IdempotencyKey:  fmt.Sprintf("chat:settle:%d", params.RequestRecord.ID),
			Reason:          "chat completion settlement",
		})
		if err != nil {
			return err
		}
	}

	_, err = txRequestLog.MarkRequestSucceeded(ctx, requestlog.MarkRequestSucceededParams{
		ID:              params.RequestRecord.ID,
		ResponseModelID: params.ResponseModelID,
		FinalProviderID: params.FinalProviderID,
		FinalChannelID:  params.FinalChannelID,
		CompletedAt:     now,
	})
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// numericIsZero 判断 NUMERIC 金额是否表示 0。
func numericIsZero(value pgtype.Numeric) bool {
	if !value.Valid || value.Int == nil {
		return true
	}
	return value.Int.Sign() == 0
}
