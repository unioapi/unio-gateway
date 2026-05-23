package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/billing"
	"github.com/ThankCat/unio-api/internal/failure"
	"github.com/ThankCat/unio-api/internal/httpapi"
	"github.com/ThankCat/unio-api/internal/ledger"
	"github.com/ThankCat/unio-api/internal/requestlog"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

const defaultAuthorizationMaxCompletionTokens int64 = 4096

// ChatAuthorizer 定义 chat 请求调用上游前冻结余额、失败后释放冻结余额的能力。
// gateway 只依赖这个边界，不直接知道 price snapshot 和 ledger reservation 的内部写法。
type ChatAuthorizer interface {
	// AuthorizeChat 在调用上游前为本次 chat 请求冻结余额。
	AuthorizeChat(ctx context.Context, params ChatAuthorizeParams) (ChatAuthorization, error)
	// ReleaseChat 在请求没有进入可扣费成功语义时释放冻结余额。
	ReleaseChat(ctx context.Context, params ChatReleaseAuthorizationParams) error
}

// ChatAuthorizeParams 表示一次 chat 请求冻结余额所需的业务事实。
// 它由 ChatCompletionService 在 request record 和 route plan 创建后组装。
type ChatAuthorizeParams struct {
	RequestRecord requestlog.RequestRecord
	Principal     *auth.APIKeyPrincipal
	Request       httpapi.ChatCompletionRequest
	ModelDBID     int64
}

// ChatAuthorization 表示一次已经创建成功的请求级资金冻结。
// ReservationID 后续交给 settlement，用来 capture 同一笔冻结资金。
type ChatAuthorization struct {
	ReservationID   int64
	RequestRecordID int64
	Amount          pgtype.Numeric
	Currency        string
}

// ChatReleaseAuthorizationParams 表示释放一次冻结余额所需参数。
// 只在没有可靠 usage、没有进入成功扣费语义时调用。
type ChatReleaseAuthorizationParams struct {
	RequestRecordID int64
	ReservationID   int64
}

// ChatAuthorizationPriceStore 定义读取当前有效价格的能力。
type ChatAuthorizationPriceStore interface {
	FindActivePriceForModel(ctx context.Context, arg sqlc.FindActivePriceForModelParams) (sqlc.Price, error)
}

// ChatAuthorizationBilling 定义冻结金额估算能力。
type ChatAuthorizationBilling interface {
	EstimateAuthorizationAmount(estimate billing.AuthorizationEstimate, price billing.PriceSnapshot) (billing.Settlement, error)
}

// ChatAuthorizationLedger 定义创建和释放余额冻结的账本能力。
type ChatAuthorizationLedger interface {
	PreAuthorize(ctx context.Context, params ledger.PreAuthorizeParams) (ledger.Reservation, error)
	Release(ctx context.Context, params ledger.ReleaseParams) (ledger.Reservation, error)
}

// ChatAuthorizationService 负责 chat 请求调用上游前的余额冻结。
type ChatAuthorizationService struct {
	priceStore ChatAuthorizationPriceStore
	billing    ChatAuthorizationBilling
	ledger     ChatAuthorizationLedger
}

// NewChatAuthorizationService 创建 chat 余额冻结 service。
func NewChatAuthorizationService(priceStore ChatAuthorizationPriceStore, billing ChatAuthorizationBilling, ledger ChatAuthorizationLedger) *ChatAuthorizationService {
	if priceStore == nil {
		panic("gateway: chat authorization price store is required")
	}
	if billing == nil {
		panic("gateway: chat authorization billing is required")
	}
	if ledger == nil {
		panic("gateway: chat authorization ledger is required")
	}

	return &ChatAuthorizationService{
		priceStore: priceStore,
		billing:    billing,
		ledger:     ledger,
	}
}

// AuthorizeChat 在调用上游前冻结本次请求的预估费用。
// 最终扣费仍以 settlement 阶段的真实 usage 为准。
func (s *ChatAuthorizationService) AuthorizeChat(ctx context.Context, params ChatAuthorizeParams) (ChatAuthorization, error) {
	now := time.Now()

	// 冻结余额只读取当前生效价格；price snapshot 留给成功 settlement 创建。
	price, err := s.priceStore.FindActivePriceForModel(ctx, sqlc.FindActivePriceForModelParams{
		ModelID: params.ModelDBID,
		AtTime:  pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return ChatAuthorization{}, failure.Wrap(
			failure.CodeGatewayChatAuthorizationFailed,
			err,
			failure.WithMessage("find active price for chat authorization"),
		)
	}

	// 这里是控损估算，不是最终计费依据。
	settlement, err := s.billing.EstimateAuthorizationAmount(
		billing.AuthorizationEstimate{
			PromptTokens:        estimatePromptTokensForAuthorization(params.Request.Messages),
			MaxCompletionTokens: estimateMaxCompletionTokens(params.Request),
		},
		billing.PriceSnapshot{
			Currency:             price.Currency,
			PricingUnit:          price.PricingUnit,
			InputPrice:           price.InputPrice,
			OutputPrice:          price.OutputPrice,
			CachedInputPrice:     price.CachedInputPrice,
			ReasoningOutputPrice: price.ReasoningOutputPrice,
			FormulaVersion:       billing.FormulaVersionV1,
		},
	)
	if err != nil {
		return ChatAuthorization{}, err
	}

	// TODO(阶段7/production): [GAP-7-014] 当前 authorization 必须全额冻结 estimated amount，低余额用户会被直接拒绝，未实现“部分冻结可用余额 + 平台差额核销”的最终产品规则；公开计费 API 前；拆分 estimated_amount 和 authorized_amount，available>0 时冻结可用余额并记录平台风险敞口。
	// 用 request_record_id 做幂等边界，避免同一请求重复冻结余额。
	reservation, err := s.ledger.PreAuthorize(ctx, ledger.PreAuthorizeParams{
		UserID:          params.Principal.UserID,
		RequestRecordID: params.RequestRecord.ID,
		Amount:          settlement.Amount,
		Currency:        settlement.Currency,
		IdempotencyKey:  fmt.Sprintf("chat:authorize:%d", params.RequestRecord.ID),
		Reason:          "chat completion authorization",
	})
	if err != nil {
		return ChatAuthorization{}, err
	}

	return ChatAuthorization{
		ReservationID:   reservation.ID,
		RequestRecordID: reservation.RequestRecordID,
		Amount:          reservation.AuthorizedAmount,
		Currency:        reservation.Currency,
	}, nil
}

// ReleaseChat 释放未进入成功结算语义的冻结余额。
func (s *ChatAuthorizationService) ReleaseChat(ctx context.Context, params ChatReleaseAuthorizationParams) error {
	reservationID := params.ReservationID

	_, err := s.ledger.Release(ctx, ledger.ReleaseParams{
		RequestRecordID: params.RequestRecordID,
		ReservationID:   &reservationID,
	})

	return err
}

// releaseChatAuthorization 脱离客户端取消上下文释放冻结余额。
func (s *ChatCompletionService) releaseChatAuthorization(ctx context.Context, authorization ChatAuthorization) error {
	if authorization.RequestRecordID == 0 || authorization.ReservationID == 0 {
		return nil
	}

	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	return s.chatAuthorizer.ReleaseChat(releaseCtx, ChatReleaseAuthorizationParams{
		RequestRecordID: authorization.RequestRecordID,
		ReservationID:   authorization.ReservationID,
	})
}

// TODO(阶段7/production): [GAP-7-013] 冻结余额时 prompt token 目前使用临时估算，可能导致冻结金额不准；接入 provider/model tokenizer 前；替换为按模型维度的 token 估算器。
func estimatePromptTokensForAuthorization(message []httpapi.ChatMessage) int64 {
	var total int64
	for _, m := range message {
		total += int64(len(m.Role))
		total += int64(len(m.Content))

		// 估算每条 message 的结构开销，后续替换为 provider/model tokenizer。
		total += 4
	}

	return total + 8
}

func estimateMaxCompletionTokens(req httpapi.ChatCompletionRequest) int64 {
	if req.MaxTokens != nil {
		return int64(*req.MaxTokens)
	}
	return defaultAuthorizationMaxCompletionTokens
}
