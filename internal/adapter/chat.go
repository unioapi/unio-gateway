package adapter

import (
	"context"

	"github.com/ThankCat/unio-api/internal/channel"
)

// ChatAdapter 定义聊天补全 adapter 需要提供的协议转换和上游调用能力。
type ChatAdapter interface {
	ChatCompletions(ctx context.Context, ch channel.Runtime, req ChatRequest) (*ChatResponse, error)
}

// StreamChatAdapter 定义聊天补全流式 adapter 能力。
type StreamChatAdapter interface {
	StreamChatCompletions(ctx context.Context, ch channel.Runtime, req ChatRequest, emit func(ChatStreamChunk) error) error
}

// ChatRequest 是 gateway 传给聊天补全 adapter 的内部请求 DTO。
type ChatRequest struct {
	Model    string
	Messages []ChatMessage
}

// ChatMessage 表示 adapter 层的单条聊天消息。
type ChatMessage struct {
	Role    string
	Content string
}

// ChatResponse 是聊天补全 adapter 返回给 gateway 的内部响应 DTO。
type ChatResponse struct {
	ID      string
	Model   string
	Content string
	Usage   ChatUsage
}

// ChatUsage 表示 adapter 从上游响应中解析出的 token 用量。
type ChatUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// ChatStreamChunk 表示 adapter 返回给 gateway 的一段聊天补全流式内容。
type ChatStreamChunk struct {
	ID           string
	Model        string
	Role         string
	Content      string
	FinishReason *string
}
