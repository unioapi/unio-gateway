package gateway

import (
	"context"
	"time"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/channel"
	"github.com/ThankCat/unio-api/internal/httpapi"
)

// ChatCompletionService 把 HTTP 层请求转换为 adapter 请求。
type ChatCompletionService struct {
	// TODO(阶段5/production): 当前直接持有 adapter 和 runtime channel 只是过渡实现；后续应接入 routing/channel selection、usage 统计、billing 和 fallback。
	adapter adapter.ChatAdapter
	channel channel.Runtime
}

// NewChatCompletionService 创建聊天补全 gateway service。
func NewChatCompletionService(adapter adapter.ChatAdapter, channel channel.Runtime) *ChatCompletionService {
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
