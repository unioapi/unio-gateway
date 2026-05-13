package gateway

import (
	"context"
	"time"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/channel"
	"github.com/ThankCat/unio-api/internal/httpapi"
)

// chatAdapter 是 ChatCompletionService 当前需要的 adapter 能力集合。
type chatAdapter interface {
	adapter.ChatAdapter
	adapter.StreamChatAdapter
}

// ChatCompletionService 把 HTTP 层请求转换为 adapter 请求。
type ChatCompletionService struct {
	// TODO(阶段6/production): 直接持有单一 runtime channel 会绕过 channel health 和同模型 fallback；实现 routing/channel selection 时；改为每次请求由 routing 选择 channel 后传给 adapter。
	adapter chatAdapter
	channel channel.Runtime
}

// NewChatCompletionService 创建聊天补全 gateway service。
func NewChatCompletionService(adapter chatAdapter, channel channel.Runtime) *ChatCompletionService {
	return &ChatCompletionService{adapter: adapter, channel: channel}
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

	adapterResp, err := s.adapter.ChatCompletions(ctx, s.channel, adapter.ChatRequest{
		Model:    req.Model,
		Messages: messages,
	})
	if err != nil {
		return nil, err
	}

	return &httpapi.ChatCompletionResponse{
		ID:      adapterResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   adapterResp.Model,
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

// StreamChatCompletion 调用 adapter 完成流式聊天补全，并转换为 HTTP stream DTO。
func (s *ChatCompletionService) StreamChatCompletion(ctx context.Context, req httpapi.ChatCompletionRequest, emit func(httpapi.ChatCompletionStreamResponse) error) error {
	messages := make([]adapter.ChatMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, adapter.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return s.adapter.StreamChatCompletions(ctx, s.channel, adapter.ChatRequest{
		Model:    req.Model,
		Messages: messages,
	}, func(chunk adapter.ChatStreamChunk) error {
		httpChunk := httpapi.ChatCompletionStreamResponse{
			ID:      chunk.ID,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   chunk.Model,
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
		}
		return emit(httpChunk)
	})
}
