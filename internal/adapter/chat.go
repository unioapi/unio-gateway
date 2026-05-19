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
// TODO(阶段5/production): [GAP-5-001] HTTP DTO 已接收 temperature/top_p/max_tokens/stop/user 等参数但 adapter contract 未承载，用户传参会被静默丢弃；开放 OpenAI-compatible chat API 前；扩展 adapter.ChatRequest 和各 provider wire DTO，或显式拒绝暂不支持的参数。
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
	// PromptTokens 是输入 prompt token 总数。
	PromptTokens int

	// CompletionTokens 是输出 completion token 总数，包含 reasoning tokens。
	CompletionTokens int

	// TotalTokens 是本次请求总 token 数，通常等于 PromptTokens + CompletionTokens。
	TotalTokens int

	// CachedTokens 是 prompt tokens 中命中上游 prompt cache 的数量。
	CachedTokens int

	// ReasoningTokens 是 completion tokens 中用于模型内部推理的数量。
	ReasoningTokens int
}

// ChatStreamChunk 表示 adapter 返回给 gateway 的一段聊天补全流式内容。
type ChatStreamChunk struct {
	ID           string
	Model        string
	Role         string
	Content      string
	FinishReason *string

	// Usage 只在 provider 返回 final usage stream chunk 时设置。
	// 普通内容 chunk 必须为 nil。
	Usage *ChatUsage
}
