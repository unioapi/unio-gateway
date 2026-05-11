package httpapi

import (
	"context"
	"time"
)

// mockChatCompletionService 是临时 chat completion 服务。
// TODO(阶段5/production): 引入 gateway/provider 后移除该服务；真实请求必须经过 provider adapter、usage 统计和 billing 前置流程。
type mockChatCompletionService struct{}

// NewMockChatCompletionService 创建临时 mock chat completion service。
func NewMockChatCompletionService() ChatCompletionService {
	return &mockChatCompletionService{}
}

// CreateChatCompletion 返回固定 mock 响应，保持当前接口可用。
func (s *mockChatCompletionService) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	return &ChatCompletionResponse{
		ID:      "chatcmpl_mock",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []ChatCompletionChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: "mock response",
				},
				FinishReason: "stop",
			},
		},
		Usage: ChatCompletionUsage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}, nil
}
