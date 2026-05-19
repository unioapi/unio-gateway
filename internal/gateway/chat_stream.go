package gateway

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/httpapi"
	"github.com/ThankCat/unio-api/internal/routing"
)

// StreamChatCompletion 编排流式 chat completion 请求，并通过 emit 写出 OpenAI-compatible SSE chunk。
func (s *ChatCompletionService) StreamChatCompletion(ctx context.Context, req httpapi.ChatCompletionRequest, emit func(httpapi.ChatCompletionStreamResponse) error) error {
	messages := make([]adapter.ChatMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, adapter.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	principal, ok := auth.APIKeyPrincipalFromContext(ctx)
	if !ok {
		return auth.ErrMissingAPIKey
	}

	// 先创建 request_records，并标记为 running。
	requestRecord, err := s.createRequestRecord(ctx, principal, req, true)
	if err != nil {
		return err
	}

	plan, err := s.router.PlanChat(ctx, routing.ChatRouteRequest{
		ProjectID: principal.ProjectID,
		ModelID:   req.Model,
	})
	if err != nil {
		s.markRequestRecordFailed(ctx, requestRecord, "routing_error", err)
		return err
	}

	var lastErr error

	for index, candidate := range plan.Candidates {
		// 每个 stream candidate 也必须先创建 attempt。
		// stream 的失败可能发生在首 chunk 前、首 chunk 后或客户端取消时，提前记录 attempt 才能审计这些状态。
		attemptRecord, err := s.createAttemptRecord(ctx, requestRecord, index, candidate)
		if err != nil {
			s.markRequestRecordFailed(ctx, requestRecord, "request_attempt_create_failed", err)
			return err
		}

		streamAdapter, ok := s.registry.StreamChat(candidate.AdapterKey)
		if !ok {
			err := fmt.Errorf("gateway: stream chat adapter %q not registered", candidate.AdapterKey)

			s.markAttemptRecordFailed(ctx, attemptRecord, "adapter_not_registered", err)
			s.markRequestRecordFailed(ctx, requestRecord, "adapter_not_registered", err)

			return err
		}

		// emitted 表示是否已经尝试向客户端写出过 SSE chunk。
		// 一旦写出开始，就不能再 fallback 到其他 channel，否则同一个 SSE 响应会混入不同上游的内容。
		emitted := false

		// finalUsage 是流式请求能否进入账务结算的唯一依据。
		// 只要上游返回 final usage，就说明本次请求已有可审计的准确 token 用量；
		// 没有 final usage 时不能猜测扣费，只能记录 failed/canceled 状态。
		var finalUsage *adapter.ChatUsage

		// upstreamResponseModel 优先使用 final usage chunk 携带的 model。
		// 如果上游 final usage chunk 没有 model，则退回 routing 选中的 upstream model。
		upstreamResponseModel := candidate.UpstreamModel

		// settleStreamFinalUsage 使用 final usage 结算流式请求。
		// stream 结算不能依赖原始请求 ctx，因为客户端可能已经断开；
		// 只要上游已经返回 final usage，平台就有准确账务事实，必须尽力完成结算。
		settleStreamFinalUsage := func() error {
			if finalUsage == nil {
				return fmt.Errorf("gateway: stream final usage is missing")
			}

			// 客户端断开会取消原始请求 ctx；结算属于服务端账务收口，
			// 不能因为客户端不再读取响应就放弃扣费。
			settlementCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()

			return s.chatSettlement.SettleSuccessfulChat(settlementCtx, ChatSettlementParams{
				RequestRecord:         requestRecord,
				AttemptRecord:         attemptRecord,
				Principal:             principal,
				ResponseModelID:       req.Model,
				ModelDBID:             candidate.ModelDBID,
				FinalProviderID:       candidate.ProviderID,
				FinalChannelID:        candidate.Channel.ID,
				UpstreamResponseModel: upstreamResponseModel,
				Usage:                 *finalUsage,
			})
		}

		// TODO(阶段7/production): [GAP-7-002] 流式请求调用上游前没有预授权，长输出或恶意断开可能让平台先承担上游成本；公开 stream 计费 API 前；基于 max_tokens/模型价格冻结余额，拿到 final usage 后 settle，多余部分 refund。
		err = streamAdapter.StreamChatCompletions(ctx, candidate.Channel, adapter.ChatRequest{
			Model:    candidate.UpstreamModel,
			Messages: messages,
		}, func(chunk adapter.ChatStreamChunk) error {
			if chunk.Usage != nil {
				// usage chunk 是 adapter 给 gateway 的内部控制事件，不是用户可见内容。
				// 这里不能设置 emitted，也不能写出 SSE，否则客户端会收到空 choices chunk。
				usage := *chunk.Usage
				finalUsage = &usage

				if chunk.Model != "" {
					upstreamResponseModel = chunk.Model
				}

				return nil
			}

			emitted = true

			return emit(httpapi.ChatCompletionStreamResponse{
				ID:      chunk.ID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []httpapi.ChatCompletionStreamChoice{
					{
						Index: 0,
						Delta: httpapi.ChatCompletionStreamDelta{
							Role:    chunk.Role,
							Content: chunk.Content,
						},
						FinishReason: chunk.FinishReason,
					},
				},
			})
		})

		if err != nil {
			// 有 final usage 时优先结算：这说明上游已经给出准确 token 用量。
			// 即使后续发生客户端取消、连接尾部错误或 adapter 返回错误，也不能让已产生成本的请求免费。
			if finalUsage != nil {
				if settleErr := settleStreamFinalUsage(); settleErr != nil {
					s.markRequestRecordFailed(ctx, requestRecord, "stream_chat_settlement_failed", settleErr)
					return settleErr
				}

				// 账务已经成功收口，但调用方仍需知道 stream 末尾发生过错误；
				// HTTP 层如果已写出 SSE，只能中断连接，不能再改写成 JSON error。
				return err
			}

			// 客户端取消不是上游失败，也不应该触发 fallback。
			// 没有 final usage 时缺少可靠用量事实，当前阶段只记录 canceled，不扣费。
			// 后续通过 pre-authorize / refund 或异常风控处理恶意取消风险。
			if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
				s.markRequestCanceled(ctx, requestRecord, attemptRecord, err)
				return err
			}

			s.markAttemptRecordFailed(ctx, attemptRecord, "stream_adapter_error", err)

			if emitted {
				// SSE 已经写出后只能把当前请求标记为失败并结束。
				// 此时 HTTP 层不能再改写普通 JSON error，也不能换 channel 重放已写出的内容。
				s.markRequestRecordFailed(ctx, requestRecord, "stream_adapter_error_after_emit", err)
				return err
			}

			// 首 chunk 前失败时，客户端还没有看到任何上游内容；只有这时才允许同模型 fallback。
			if !s.retryClassifier.IsRetryable(err) {
				s.markRequestRecordFailed(ctx, requestRecord, "stream_adapter_error", err)
				return err
			}

			lastErr = err
			continue
		}

		if finalUsage == nil {
			// adapter 正常结束但没有 final usage，不能把它当作可计费成功请求。
			// 这类请求可能是上游不支持 include_usage、代理吞掉尾包，或 parser 漏解析。
			err := fmt.Errorf("gateway: stream final usage is missing")
			s.markAttemptRecordFailed(ctx, attemptRecord, "stream_usage_missing", err)
			s.markRequestRecordFailed(ctx, requestRecord, "stream_usage_missing", err)
			return err
		}

		if settleErr := settleStreamFinalUsage(); settleErr != nil {
			s.markRequestRecordFailed(ctx, requestRecord, "stream_chat_settlement_failed", settleErr)
			return settleErr
		}

		return nil
	}

	if lastErr != nil {
		s.markRequestRecordFailed(ctx, requestRecord, "stream_adapter_error", lastErr)
		return lastErr
	}

	err = routing.ErrNoAvailableChannel
	s.markRequestRecordFailed(ctx, requestRecord, "no_available_channel", err)
	return err
}
