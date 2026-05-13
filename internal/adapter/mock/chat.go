package mock

import (
	"context"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/channel"
)

// chatAdapter 是临时 mock adapter。
// TODO(阶段6/production): mock adapter 会掩盖真实 channel selection 和上游错误；接入 model catalog 和 routing 时；移除 mock adapter 并由 routing 选择真实 adapter/channel。
type chatAdapter struct{}

// NewChatAdapter 创建临时 mock adapter。
func NewChatAdapter() *chatAdapter {
	return &chatAdapter{}
}

// ChatCompletions 返回固定 mock 响应，保持当前接口可用。
func (a *chatAdapter) ChatCompletions(ctx context.Context, runtime channel.Runtime, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
	return &adapter.ChatResponse{
		ID:      "chatcmpl_mock",
		Model:   req.Model,
		Content: "mock response",
	}, nil
}

// StreamChatCompletions 返回固定 mock 流式响应，保持当前 stream 接口可用。
func (a *chatAdapter) StreamChatCompletions(ctx context.Context, ch channel.Runtime, req adapter.ChatRequest, emit func(chunk adapter.ChatStreamChunk) error) error {
	if err := emit(adapter.ChatStreamChunk{
		ID:           "chatcmpl_mock",
		Model:        req.Model,
		Role:         "assistant",
		Content:      "mock response",
		FinishReason: nil,
	}); err != nil {
		return err
	}

	return nil

}
