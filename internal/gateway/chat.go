package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/httpapi"
	"github.com/ThankCat/unio-api/internal/httpx"
	"github.com/ThankCat/unio-api/internal/requestlog"
	"github.com/ThankCat/unio-api/internal/routing"
)

// ChatRouter 定义 gateway 生成 chat route plan 所需的 routing 能力。
type ChatRouter interface {
	PlanChat(ctx context.Context, req routing.ChatRouteRequest) (routing.ChatRoutePlan, error)
}

// AdapterRegistry 定义 gateway 根据 adapter key 查找 adapter 的能力。
type AdapterRegistry interface {
	Chat(adapterKey string) (adapter.ChatAdapter, bool)
	StreamChat(adapterKey string) (adapter.StreamChatAdapter, bool)
}

// RetryClassifier 定义 gateway 判断错误是否允许尝试下一个同模型 channel 的能力。
type RetryClassifier interface {
	IsRetryable(err error) bool
}

// ChatSettlementExecutor 定义非流式 chat 成功后提交结算事务的能力。
type ChatSettlementExecutor interface {
	SettleSuccessfulChat(ctx context.Context, params ChatSettlementParams) error
}

// ChatCompletionService 把 HTTP 层请求转换为 adapter 请求。
type ChatCompletionService struct {
	router          ChatRouter
	registry        AdapterRegistry
	retryClassifier RetryClassifier
	requestLog      requestlog.Service
	chatSettlement  ChatSettlementExecutor
}

// NewChatCompletionService 创建聊天补全 gateway service。
func NewChatCompletionService(
	router ChatRouter,
	registry AdapterRegistry,
	retryClassifier RetryClassifier,
	requestLog requestlog.Service,
	chatSettlement ChatSettlementExecutor,
) *ChatCompletionService {
	if retryClassifier == nil {
		retryClassifier = NeverRetryClassifier{}
	}
	if requestLog == nil {
		panic("gateway: request log service is required")
	}
	if chatSettlement == nil {
		panic("gateway: chat settlement service is required")
	}

	return &ChatCompletionService{
		router:          router,
		registry:        registry,
		retryClassifier: retryClassifier,
		requestLog:      requestLog,
		chatSettlement:  chatSettlement,
	}
}

// createRequestRecord 创建用户可见请求记录，并立即推进到 running 状态。
func (s *ChatCompletionService) createRequestRecord(
	ctx context.Context,
	principal *auth.APIKeyPrincipal,
	req httpapi.ChatCompletionRequest,
	stream bool,
) (requestlog.RequestRecord, error) {
	// TODO(阶段7/production): request_records.request_id 复用可由客户端传入的 X-Request-ID 会导致重复 header 触发唯一约束；开放公网 API 前；拆分服务端生成 request_id 和 trace/correlation id，request record 只保存服务端生成 ID。
	requestID := httpx.RequestID(ctx)
	if requestID == "" {
		return requestlog.RequestRecord{}, fmt.Errorf("gateway: request id missing")
	}

	record, err := s.requestLog.CreateRequest(ctx, requestlog.CreateRequestParams{
		RequestID:        requestID,
		UserID:           principal.UserID,
		ProjectID:        principal.ProjectID,
		APIKeyID:         principal.APIKeyID,
		RequestedModelID: req.Model,
		Stream:           stream,
		StartedAt:        time.Now(),
	})
	if err != nil {
		return requestlog.RequestRecord{}, err
	}

	record, err = s.requestLog.MarkRequestRunning(ctx, record.ID)
	if err != nil {
		return requestlog.RequestRecord{}, err
	}

	return record, nil
}

// markStreamRequestSucceeded 把流式请求标记为成功，并记录最终 provider/channel。
func (s *ChatCompletionService) markStreamAttemptSucceeded(
	ctx context.Context,
	attempt requestlog.AttemptRecord,
	upstreamModel string,
) error {
	_, err := s.requestLog.MarkAttemptSucceeded(ctx, requestlog.MarkAttemptSucceededParams{
		ID:                    attempt.ID,
		UpstreamResponseModel: upstreamModel,
		UpstreamStatusCode:    200,
		UpstreamRequestID:     nil,
		CompletedAt:           time.Now(),
	})
	return err
}

// markRequestFailed 把 request record 标记为失败；失败路径不覆盖原始业务错误。
func (s *ChatCompletionService) markRequestFailed(
	ctx context.Context,
	requestRecord requestlog.RequestRecord,
	code string,
	err error,
) {
	message := ""
	if err != nil {
		message = err.Error()
	}

	_, _ = s.requestLog.MarkRequestFailed(ctx, requestlog.MarkRequestFailedParams{
		ID:           requestRecord.ID,
		ErrorCode:    code,
		ErrorMessage: message,
		CompletedAt:  time.Now(),
	})
}

// createAttempt 创建一次上游 channel 尝试记录。
func (s *ChatCompletionService) createAttempt(
	ctx context.Context,
	requestRecord requestlog.RequestRecord,
	attemptIndex int,
	candidate routing.ChatRouteCandidate,
) (requestlog.AttemptRecord, error) {
	attempt, err := s.requestLog.CreateAttempt(ctx, requestlog.CreateAttemptParams{
		RequestRecordID: requestRecord.ID,
		AttemptIndex:    attemptIndex,
		ProviderID:      candidate.ProviderID,
		ChannelID:       candidate.Channel.ID,
		AdapterKey:      candidate.AdapterKey,
		UpstreamModel:   candidate.UpstreamModel,
		StartedAt:       time.Now(),
	})
	if err != nil {
		return requestlog.AttemptRecord{}, err
	}

	return attempt, nil
}

// markAttemptFailed 把一次上游尝试标记为失败；失败路径不覆盖原始业务错误。
func (s *ChatCompletionService) markAttemptFailed(
	ctx context.Context,
	attempt requestlog.AttemptRecord,
	code string,
	err error,
) {
	message := ""
	if err != nil {
		message = err.Error()
	}

	_, _ = s.requestLog.MarkAttemptFailed(ctx, requestlog.MarkAttemptFailedParams{
		ID:           attempt.ID,
		ErrorCode:    code,
		ErrorMessage: message,
		CompletedAt:  time.Now(),
	})
}

// markStreamRequestSucceeded 把流式请求标记为成功，并记录最终 provider/channel。
func (s *ChatCompletionService) markStreamRequestSucceeded(
	ctx context.Context,
	requestRecord requestlog.RequestRecord,
	req httpapi.ChatCompletionRequest,
	candidate routing.ChatRouteCandidate,
) error {
	_, err := s.requestLog.MarkRequestSucceeded(ctx, requestlog.MarkRequestSucceededParams{
		ID:              requestRecord.ID,
		ResponseModelID: req.Model,
		FinalProviderID: candidate.ProviderID,
		FinalChannelID:  candidate.Channel.ID,
		CompletedAt:     time.Now(),
	})
	return err
}

// CreateChatCompletion 调用 adapter 完成聊天补全，并转换为 HTTP 响应 DTO。
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
		return nil, auth.ErrMissingAPIKey
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
		s.markRequestFailed(ctx, requestRecord, "routing_error", err)
		return nil, err
	}

	var lastErr error

	for index, candidate := range plan.Candidates {
		// 每个 candidate 都先创建 attempt，再调用 adapter，确保 fallback 链路可审计。
		attempt, err := s.createAttempt(ctx, requestRecord, index, candidate)
		if err != nil {
			s.markRequestFailed(ctx, requestRecord, "request_attempt_create_failed", err)
			return nil, err
		}

		chatAdapter, ok := s.registry.Chat(candidate.AdapterKey)
		if !ok {
			err := fmt.Errorf("gateway: chat adapter %q not registered", candidate.AdapterKey)

			s.markAttemptFailed(ctx, attempt, "adapter_not_registered", err)
			s.markRequestFailed(ctx, requestRecord, "adapter_not_registered", err)

			return nil, err
		}

		adapterResp, err := chatAdapter.ChatCompletions(ctx, candidate.Channel, adapter.ChatRequest{
			Model:    candidate.UpstreamModel,
			Messages: messages,
		})
		if err != nil {
			s.markAttemptFailed(ctx, attempt, "adapter_error", err)

			if !s.retryClassifier.IsRetryable(err) {
				s.markRequestFailed(ctx, requestRecord, "adapter_error", err)

				return nil, err
			}
			lastErr = err
			continue
		}

		if err := s.chatSettlement.SettleSuccessfulChat(ctx, ChatSettlementParams{
			RequestRecord:   requestRecord,
			AttemptRecord:   attempt,
			Principal:       principal,
			ResponseModelID: req.Model,
			ModelDBID:       candidate.ModelDBID,
			FinalProviderID: candidate.ProviderID,
			FinalChannelID:  candidate.Channel.ID,
			AdapterResp:     adapterResp,
		}); err != nil {
			s.markRequestFailed(ctx, requestRecord, "chat_settlement_failed", err)
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
		s.markRequestFailed(ctx, requestRecord, "adapter_error", lastErr)
		return nil, lastErr
	}

	err = routing.ErrNoAvailableChannel
	s.markRequestFailed(ctx, requestRecord, "no_available_channel", err)

	return nil, err
}

// StreamChatCompletion 调用 adapter 完成流式聊天补全，并转换为 HTTP stream DTO。
func (s *ChatCompletionService) StreamChatCompletion(ctx context.Context, req httpapi.ChatCompletionRequest, emit func(httpapi.ChatCompletionStreamResponse) error) error {
	// TODO(阶段7/production): stream 请求尚未写入 request/attempt 状态会影响流式中断、结算和退款审计；进入 7.8 stream 计费小节时；按 emit 前/emit 后状态机接入 requestlog，并在可靠 final usage 可用后再 settle。
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

	requestRecord, err := s.createRequestRecord(ctx, principal, req, true)
	if err != nil {
		return err
	}

	plan, err := s.router.PlanChat(ctx, routing.ChatRouteRequest{
		ProjectID: principal.ProjectID,
		ModelID:   req.Model,
	})
	if err != nil {
		s.markRequestFailed(ctx, requestRecord, "routing_error", err)
		return err
	}

	var lastErr error

	for index, candidate := range plan.Candidates {
		// 每个 stream candidate 也必须先创建 attempt，确保 fallback 和中断都能审计。
		attempt, err := s.createAttempt(ctx, requestRecord, index, candidate)
		if err != nil {
			s.markRequestFailed(ctx, requestRecord, "request_attempt_create_failed", err)
			return err
		}

		streamAdapter, ok := s.registry.StreamChat(candidate.AdapterKey)
		if !ok {
			err := fmt.Errorf("gateway: stream chat adapter %q not registered", candidate.AdapterKey)

			s.markAttemptFailed(ctx, attempt, "adapter_not_registered", err)
			s.markRequestFailed(ctx, requestRecord, "adapter_not_registered", err)

			return err
		}

		emitted := false

		err = streamAdapter.StreamChatCompletions(ctx, candidate.Channel, adapter.ChatRequest{
			Model:    candidate.UpstreamModel,
			Messages: messages,
		}, func(chunk adapter.ChatStreamChunk) error {
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
			s.markAttemptFailed(ctx, attempt, "stream_adapter_error", err)

			if emitted {
				// SSE 已经写出后不能 fallback，否则一个客户端响应会混入两个 channel 的内容。
				s.markRequestFailed(ctx, requestRecord, "stream_adapter_error_after_emit", err)
				return err
			}

			if !s.retryClassifier.IsRetryable(err) {
				s.markRequestFailed(ctx, requestRecord, "stream_adapter_error", err)
				return err
			}

			lastErr = err
			continue
		}

		// TODO(阶段7/production): stream final usage 暂不可可靠获得会导致无法准确扣费；接入 provider stream usage 解析前；扩展 adapter stream final usage 事件并复用 settlement 事务结算。
		if err := s.markStreamAttemptSucceeded(ctx, attempt, candidate.UpstreamModel); err != nil {
			s.markRequestFailed(ctx, requestRecord, "stream_attempt_mark_succeeded_failed", err)
			return err
		}

		if err := s.markStreamRequestSucceeded(ctx, requestRecord, req, candidate); err != nil {
			s.markRequestFailed(ctx, requestRecord, "stream_request_mark_succeeded_failed", err)
			return err
		}

		return nil
	}

	if lastErr != nil {
		s.markRequestFailed(ctx, requestRecord, "stream_adapter_error", lastErr)
		return lastErr
	}

	err = routing.ErrNoAvailableChannel
	s.markRequestFailed(ctx, requestRecord, "no_available_channel", err)
	return err
}
