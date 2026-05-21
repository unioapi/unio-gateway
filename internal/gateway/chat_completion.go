package gateway

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/failure"
	"github.com/ThankCat/unio-api/internal/httpapi"
	"github.com/ThankCat/unio-api/internal/routing"
)

// CreateChatCompletion 编排非流式 chat completion 请求，并返回 OpenAI-compatible HTTP DTO。
func (s *ChatCompletionService) CreateChatCompletion(ctx context.Context, req httpapi.ChatCompletionRequest) (*httpapi.ChatCompletionResponse, error) {
	messages := make([]adapter.ChatMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, adapter.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	principal, ok := auth.APIKeyPrincipalFromContext(ctx)
	if !ok {
		return nil, failure.Wrap(
			failure.CodeAuthMissingAPIKey,
			auth.ErrMissingAPIKey,
			failure.WithMessage(auth.ErrMissingAPIKey.Error()),
		)
	}

	requestRecord, err := s.createRequestRecord(ctx, principal, req, false)
	if err != nil {
		return nil, err
	}

	plan, err := s.router.PlanChat(ctx, routing.ChatRouteRequest{
		ProjectID: principal.ProjectID,
		ModelID:   req.Model,
	})
	if err != nil {
		s.markRequestRecordFailed(ctx, requestRecord, routingFailureCode(err), err)
		return nil, err
	}

	var lastErr error

	for index, candidate := range plan.Candidates {
		// 每个 candidate 都先创建 attempt，再调用 adapter。
		// 这样即使后续 fallback，也能在 request_attempts 里还原完整尝试链路。
		attemptRecord, err := s.createAttemptRecord(ctx, requestRecord, index, candidate)
		if err != nil {
			s.markRequestRecordFailed(ctx, requestRecord, "request_attempt_create_failed", err)
			return nil, err
		}

		chatAdapter, ok := s.registry.Chat(candidate.AdapterKey)
		if !ok {
			err := failure.New(
				failure.CodeGatewayAdapterNotRegistered,
				failure.WithMessage(fmt.Sprintf("gateway chat adapter %q not registered", candidate.AdapterKey)),
			)

			s.markAttemptRecordFailed(ctx, attemptRecord, "adapter_not_registered", err)
			s.markRequestRecordFailed(ctx, requestRecord, "adapter_not_registered", err)

			return nil, err
		}

		// TODO(阶段7/production): [GAP-7-001] 非流式请求调用上游前没有余额预检或预授权，余额不足用户可能先产生平台上游成本再在 settlement 阶段失败；公开计费 API 前；引入余额 preflight 或 pre-authorize，并在 settlement 成功后确认扣费。
		adapterResp, err := chatAdapter.ChatCompletions(ctx, candidate.Channel, adapter.ChatRequest{
			Model:            candidate.UpstreamModel,
			Messages:         messages,
			Temperature:      req.Temperature,
			TopP:             req.TopP,
			MaxTokens:        req.MaxTokens,
			PresencePenalty:  req.PresencePenalty,
			FrequencyPenalty: req.FrequencyPenalty,
			Stop:             req.Stop,
			User:             req.User,
		})
		if err != nil {
			// 客户端取消不是上游失败，也不应该触发 fallback。
			// 此时还没有进入 settlement，不会写 usage、price snapshot 或 ledger。
			if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
				s.markRequestCanceled(ctx, requestRecord, attemptRecord, err)
				return nil, err
			}

			s.markAttemptRecordFailed(ctx, attemptRecord, "adapter_error", err)

			if !s.retryClassifier.IsRetryable(err) {
				s.markRequestRecordFailed(ctx, requestRecord, "adapter_error", err)

				return nil, err
			}
			lastErr = err
			continue
		}

		// 非流式成功请求的账务事实必须在 settlement 事务内一起提交。
		// 这里不能先返回 HTTP response 再异步扣费，否则 usage、price snapshot、ledger 和 request status 会出现不一致窗口。
		if err := s.chatSettlement.SettleSuccessfulChat(ctx, ChatSettlementParams{
			RequestRecord:         requestRecord,
			AttemptRecord:         attemptRecord,
			Principal:             principal,
			ResponseModelID:       req.Model,
			ModelDBID:             candidate.ModelDBID,
			FinalProviderID:       candidate.ProviderID,
			FinalChannelID:        candidate.Channel.ID,
			UpstreamResponseModel: adapterResp.Model,
			Usage:                 adapterResp.Usage,
		}); err != nil {
			s.markRequestRecordFailed(ctx, requestRecord, "chat_settlement_failed", err)
			return nil, err
		}

		return &httpapi.ChatCompletionResponse{
			ID:      adapterResp.ID,
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []httpapi.ChatCompletionChoice{
				{
					Index: 0,
					Message: httpapi.ChatMessage{
						Role:    "assistant",
						Content: adapterResp.Content,
					},
					FinishReason: "stop",
				},
			},
			Usage: httpapi.ChatCompletionUsage{
				PromptTokens:     adapterResp.Usage.PromptTokens,
				CompletionTokens: adapterResp.Usage.CompletionTokens,
				TotalTokens:      adapterResp.Usage.TotalTokens,
			},
		}, nil
	}

	if lastErr != nil {
		s.markRequestRecordFailed(ctx, requestRecord, "adapter_error", lastErr)
		return nil, lastErr
	}

	err = failure.Wrap(
		failure.CodeRoutingNoAvailableChannel,
		routing.ErrNoAvailableChannel,
		failure.WithMessage(routing.ErrNoAvailableChannel.Error()),
	)
	s.markRequestRecordFailed(ctx, requestRecord, routingFailureCode(err), err)

	return nil, err
}
