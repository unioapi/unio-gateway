package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/httpapi"
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

// NeverRetryClassifier 是保守的错误分类器，默认不重试任何错误。
type NeverRetryClassifier struct{}

// IsRetryable 始终返回 false，避免没有明确错误分类时误触发 fallback。
func (NeverRetryClassifier) IsRetryable(err error) bool {
	return false
}

// ChatCompletionService 把 HTTP 层请求转换为 adapter 请求。
type ChatCompletionService struct {
	router          ChatRouter
	registry        AdapterRegistry
	retryClassifier RetryClassifier
}

// NewChatCompletionService 创建聊天补全 gateway service。
func NewChatCompletionService(router ChatRouter, registry AdapterRegistry, retryClassifier RetryClassifier) *ChatCompletionService {
	if retryClassifier == nil {
		retryClassifier = NeverRetryClassifier{}
	}

	return &ChatCompletionService{
		router:          router,
		registry:        registry,
		retryClassifier: retryClassifier,
	}
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

	plan, err := s.router.PlanChat(ctx, routing.ChatRouteRequest{
		ProjectID: principal.ProjectID,
		ModelID:   req.Model,
	})
	if err != nil {
		return nil, err
	}

	var lastErr error

	for _, candidate := range plan.Candidates {
		chatAdapter, ok := s.registry.Chat(candidate.AdapterKey)
		if !ok {
			return nil, fmt.Errorf("gateway: chat adapter %q not registered", candidate.AdapterKey)
		}

		adapterResp, err := chatAdapter.ChatCompletions(ctx, candidate.Channel, adapter.ChatRequest{
			Model:    candidate.UpstreamModel,
			Messages: messages,
		})
		if err != nil {
			if !s.retryClassifier.IsRetryable(err) {
				return nil, err
			}
			lastErr = err
			continue
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
		return nil, lastErr
	}

	return nil, routing.ErrNoAvailableChannel
}

// StreamChatCompletion 调用 adapter 完成流式聊天补全，并转换为 HTTP stream DTO。
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

	plan, err := s.router.PlanChat(ctx, routing.ChatRouteRequest{
		ProjectID: principal.ProjectID,
		ModelID:   req.Model,
	})
	if err != nil {
		return err
	}

	var lastErr error

	for _, candidate := range plan.Candidates {
		streamAdapter, ok := s.registry.StreamChat(candidate.AdapterKey)
		if !ok {
			return fmt.Errorf("gateway: stream chat adapter %q not registered", candidate.AdapterKey)
		}

		emitted := false

		err := streamAdapter.StreamChatCompletions(ctx, candidate.Channel, adapter.ChatRequest{
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
			if emitted {
				return err
			}

			if !s.retryClassifier.IsRetryable(err) {
				return err
			}

			lastErr = err
			continue
		}

		return nil
	}

	if lastErr != nil {
		return lastErr
	}

	return routing.ErrNoAvailableChannel
}
