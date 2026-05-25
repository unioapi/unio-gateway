package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/billing"
	"github.com/ThankCat/unio-api/internal/failure"
	"github.com/ThankCat/unio-api/internal/ledger"
	"github.com/ThankCat/unio-api/internal/requestlog"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
)

// ChatTxBeginner 定义 chat settlement 开启数据库事务所需能力。
type ChatTxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// ChatLedgerCapturer 定义 chat settlement 确认扣费或释放冻结余额所需能力。
type ChatLedgerCapturer interface {
	CaptureWithQueries(ctx context.Context, queries *sqlc.Queries, params ledger.CaptureParams) (ledger.Reservation, error)
	ReleaseWithQueries(ctx context.Context, queries *sqlc.Queries, params ledger.ReleaseParams) (ledger.Reservation, error)
}

// ChatBillingCalculator 定义 chat settlement 计算请求金额所需能力。
type ChatBillingCalculator interface {
	Calculate(usage billing.Usage, price billing.PriceSnapshot) (billing.Settlement, error)
}

// ChatSettlementService 负责 chat 请求成功后的 usage、price snapshot 和 ledger 结算。
type ChatSettlementService struct {
	db                ChatTxBeginner
	queries           *sqlc.Queries
	billingCalculator ChatBillingCalculator
	ledgerCapturer    ChatLedgerCapturer
}

// NewChatSettlementService 创建 chat 请求结算 service。
func NewChatSettlementService(db ChatTxBeginner, queries *sqlc.Queries, billingCalculator ChatBillingCalculator, ledgerCapturer ChatLedgerCapturer) *ChatSettlementService {
	if db == nil {
		panic("gateway: chat settlement tx beginner is required")
	}
	if queries == nil {
		panic("gateway: chat settlement queries is required")
	}
	if billingCalculator == nil {
		panic("gateway: chat billing calculator is required")
	}
	if ledgerCapturer == nil {
		panic("gateway: chat ledger capturer is required")
	}

	return &ChatSettlementService{
		db:                db,
		queries:           queries,
		billingCalculator: billingCalculator,
		ledgerCapturer:    ledgerCapturer,
	}
}

// ChatSettlementParams 表示一次成功 chat 请求结算所需的事实。
// 非流式 usage 来自 adapter.ChatResponse；流式 usage 来自 final usage stream chunk。
type ChatSettlementParams struct {
	RequestRecord         requestlog.RequestRecord
	AttemptRecord         requestlog.AttemptRecord
	Principal             *auth.APIKeyPrincipal
	Authorization         ChatAuthorization
	ResponseModelID       string
	ModelDBID             int64
	FinalProviderID       int64
	FinalChannelID        int64
	UpstreamResponseModel string
	Usage                 adapter.ChatUsage
}

// SettleSuccessfulChat 对一次成功的 chat 请求执行结算。
func (s *ChatSettlementService) SettleSuccessfulChat(ctx context.Context, params ChatSettlementParams) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return failure.Wrap(
			failure.CodeGatewayChatSettlementFailed,
			err,
			failure.WithMessage("begin chat settlement transaction"),
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	now := time.Now()
	txQueries := s.queries.WithTx(tx)
	txRequestLog := requestlog.NewStore(txQueries)
	usage := params.Usage

	// TODO(阶段7/production): [GAP-7-007] settlement 缺少请求级幂等完成检测，重复补偿或并发 settlement 可能撞上 usage/price snapshot 唯一约束并把已成功请求误标失败；引入补偿任务或并发重试前；按 request_record_id 检测既有 usage/snapshot/ledger 并返回幂等成功。
	// TODO(阶段8/production): [GAP-8-001] adapter 成功响应缺少真实 upstream status/request id 会影响渠道审计精度；接入 provider error classification / adapter metadata 时；从 adapter response metadata 写入真实 upstream_status_code 和 upstream_request_id。
	_, err = txRequestLog.MarkAttemptSucceeded(ctx, requestlog.MarkAttemptSucceededParams{
		ID:                    params.AttemptRecord.ID,
		UpstreamResponseModel: params.UpstreamResponseModel,
		UpstreamStatusCode:    200,
		UpstreamRequestID:     nil,
		CompletedAt:           now,
	})
	if err != nil {
		return failure.Wrap(
			failure.CodeGatewayChatSettlementFailed,
			err,
			failure.WithMessage("create usage record"),
		)
	}

	// TODO(阶段7/production): [GAP-7-008] usage_records.source 当前无法区分非流式 response 和 stream final usage，会降低账单审计与异常排查精度；收口 stream billing 报表前；在 ChatSettlementParams 中显式传入 usage source。
	_, err = txQueries.CreateUsageRecord(ctx, sqlc.CreateUsageRecordParams{
		RequestRecordID:  params.RequestRecord.ID,
		PromptTokens:     int64(usage.PromptTokens),
		CompletionTokens: int64(usage.CompletionTokens),
		TotalTokens:      int64(usage.TotalTokens),
		CachedTokens:     int64(usage.CachedTokens),
		ReasoningTokens:  int64(usage.ReasoningTokens),
		Source:           "upstream_response",
	})
	if err != nil {
		return failure.Wrap(
			failure.CodeGatewayChatSettlementFailed,
			err,
			failure.WithMessage("find active price for model"),
		)
	}

	if params.Authorization.PriceID <= 0 {
		return failure.New(
			failure.CodeGatewayChatSettlementFailed,
			failure.WithMessage("chat settlement missing authorization price id"),
		)
	}

	authorizationPrice := params.Authorization.Price

	// 将 authorization 使用的价格复制成请求级快照，保证冻结和结算使用同一份价格。
	snapshot, err := txQueries.CreatePriceSnapshot(ctx, sqlc.CreatePriceSnapshotParams{
		RequestRecordID:      params.RequestRecord.ID,
		PriceID:              pgtype.Int8{Int64: params.Authorization.PriceID, Valid: true},
		Currency:             authorizationPrice.Currency,
		PricingUnit:          authorizationPrice.PricingUnit,
		InputPrice:           authorizationPrice.InputPrice,
		OutputPrice:          authorizationPrice.OutputPrice,
		CachedInputPrice:     authorizationPrice.CachedInputPrice,
		ReasoningOutputPrice: authorizationPrice.ReasoningOutputPrice,
		FormulaVersion:       authorizationPrice.FormulaVersion,
	})
	if err != nil {
		return err
	}

	// billing 只做纯金额计算；它不写 usage、不查价格、不扣余额。
	settlement, err := s.billingCalculator.Calculate(billing.Usage{
		PromptTokens:     int64(usage.PromptTokens),
		CompletionTokens: int64(usage.CompletionTokens),
		TotalTokens:      int64(usage.TotalTokens),
		CachedTokens:     int64(usage.CachedTokens),
		ReasoningTokens:  int64(usage.ReasoningTokens),
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

	reservationID := params.Authorization.ReservationID

	// ledger_entries.amount 要求大于 0；零金额请求保留 usage 和 price snapshot，但不写余额流水。
	if numericIsZero(settlement.Amount) {
		_, err := s.ledgerCapturer.ReleaseWithQueries(ctx, txQueries, ledger.ReleaseParams{
			RequestRecordID: params.RequestRecord.ID,
			ReservationID:   &reservationID,
		})
		if err != nil {
			return err
		}
	} else {
		_, err = s.ledgerCapturer.CaptureWithQueries(ctx, txQueries, ledger.CaptureParams{
			RequestRecordID: params.RequestRecord.ID,
			ReservationID:   &reservationID,
			ActualAmount:    settlement.Amount,
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

	if err := tx.Commit(ctx); err != nil {
		return failure.Wrap(
			failure.CodeGatewayChatSettlementFailed,
			err,
			failure.WithMessage("commit chat settlement transaction"),
		)
	}

	return nil
}

// numericIsZero 判断 NUMERIC 金额是否表示 0。
func numericIsZero(value pgtype.Numeric) bool {
	if !value.Valid || value.Int == nil {
		return true
	}
	return value.Int.Sign() == 0
}
