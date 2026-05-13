package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/channel"
	"github.com/ThankCat/unio-api/internal/httpapi"
)

// fakeChatAdapter 是 gateway 测试使用的 adapter 替身。
type fakeChatAdapter struct {
	chatCalled   bool
	chatReq      adapter.ChatRequest
	chatResp     *adapter.ChatResponse
	chatErr      error
	streamCalled bool
	streamReq    adapter.ChatRequest
	streamResp   []adapter.ChatStreamChunk
	streamErr    error
	ch           channel.Runtime
}

// ChatCompletions 记录 gateway 传入的请求，并返回测试预设响应。
func (a *fakeChatAdapter) ChatCompletions(ctx context.Context, ch channel.Runtime, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
	a.chatCalled = true
	a.chatReq = req
	a.ch = ch

	return a.chatResp, a.chatErr
}

// StreamChatCompletions 记录 gateway 传入的流式请求，并返回测试预设 chunk。
func (a *fakeChatAdapter) StreamChatCompletions(ctx context.Context, ch channel.Runtime, req adapter.ChatRequest) ([]adapter.ChatStreamChunk, error) {
	a.streamCalled = true
	a.streamReq = req
	a.ch = ch

	return a.streamResp, a.streamErr
}

func TestChatCompletionServiceCreateChatCompletionCallsAdapter(t *testing.T) {
	selectedChannel := channel.Runtime{
		ID:      123,
		BaseURL: "https://example.test/v1",
		APIKey:  "test-secret",
		Timeout: 30 * time.Second,
	}
	fakeAdapter := &fakeChatAdapter{
		chatResp: &adapter.ChatResponse{
			ID:      "chatcmpl_provider_test",
			Model:   "openai/gpt-4.1",
			Content: "adapter response",
			Usage: adapter.ChatUsage{
				PromptTokens:     10,
				CompletionTokens: 11,
				TotalTokens:      21,
			},
		},
	}
	service := NewChatCompletionService(fakeAdapter, selectedChannel)

	req := httpapi.ChatCompletionRequest{
		Model: "openai/gpt-4.1",
		Messages: []httpapi.ChatMessage{
			{
				Role:    "user",
				Content: "hello",
			},
		},
	}

	got, err := service.CreateChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateChatCompletion returned err: %v", err)
	}

	if !fakeAdapter.chatCalled {
		t.Fatal("expected adapter to be called")
	}

	if fakeAdapter.chatReq.Model != "openai/gpt-4.1" {
		t.Fatalf("expected model %q, got %q", "openai/gpt-4.1", fakeAdapter.chatReq.Model)
	}

	if len(fakeAdapter.chatReq.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(fakeAdapter.chatReq.Messages))
	}

	if len(got.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(got.Choices))
	}

	if fakeAdapter.chatReq.Messages[0].Content != "hello" {
		t.Fatalf("expected message content %q, got %q", "hello", fakeAdapter.chatReq.Messages[0].Content)
	}

	if fakeAdapter.chatReq.Messages[0].Role != "user" {
		t.Fatalf("expected message role %q, got %q", "user", fakeAdapter.chatReq.Messages[0].Role)
	}

	if got.Choices[0].Message.Content != "adapter response" {
		t.Fatalf("expected content %q, got %q", "adapter response", got.Choices[0].Message.Content)
	}

	if got.ID != "chatcmpl_provider_test" {
		t.Fatalf("expected ID %q, got %q", "chatcmpl_provider_test", got.ID)
	}

	if got.Model != "openai/gpt-4.1" {
		t.Fatalf("expected model %q, got %q", "openai/gpt-4.1", got.Model)
	}

	if got.Object != "chat.completion" {
		t.Fatalf("expected object %q, got %q", "chat.completion", got.Object)
	}

	if got.Choices[0].Message.Role != "assistant" {
		t.Fatalf("expected role %q, got %q", "assistant", got.Choices[0].Message.Role)
	}

	if got.Choices[0].FinishReason != "stop" {
		t.Fatalf("expected reason %q, got %q", "stop", got.Choices[0].FinishReason)
	}

	if fakeAdapter.ch.ID != 123 {
		t.Fatalf("expected channel id %d, got %d", int64(123), fakeAdapter.ch.ID)
	}

	if fakeAdapter.ch.BaseURL != "https://example.test/v1" {
		t.Fatalf("expected channel base url %q, got %q", "https://example.test/v1", fakeAdapter.ch.BaseURL)
	}

	if fakeAdapter.ch.Timeout != 30*time.Second {
		t.Fatalf("expected channel timeout %v, got %v", 30*time.Second, fakeAdapter.ch.Timeout)
	}

	if got.Usage.PromptTokens != 10 {
		t.Fatalf("expected prompt_tokens %d, got %d", 10, got.Usage.PromptTokens)
	}
	if got.Usage.CompletionTokens != 11 {
		t.Fatalf("expected completion_tokens %d, got %d", 11, got.Usage.CompletionTokens)
	}
	if got.Usage.TotalTokens != 21 {
		t.Fatalf("expected total_tokens %d, got %d", 21, got.Usage.TotalTokens)
	}
}

func TestChatCompletionServiceStreamChatCompletionCallsAdapter(t *testing.T) {
	fakeAdapter := &fakeChatAdapter{
		streamResp: []adapter.ChatStreamChunk{
			{
				ID:      "chatcmpl_mock",
				Model:   "openai/gpt-4.1",
				Role:    "assistant",
				Content: "mock response",
			},
		},
	}
	selectedChannel := channel.Runtime{
		ID:      123,
		BaseURL: "https://example.test/v1",
		APIKey:  "test-secret",
		Timeout: 30 * time.Second,
	}

	service := NewChatCompletionService(fakeAdapter, selectedChannel)

	req := httpapi.ChatCompletionRequest{
		Model:    "openai/gpt-4.1",
		Messages: []httpapi.ChatMessage{{Role: "user", Content: "hello"}},
	}

	chunks, err := service.StreamChatCompletion(context.Background(), req)

	if !fakeAdapter.streamCalled {
		t.Fatal("expected adapter to be called")
	}

	if fakeAdapter.streamReq.Model != "openai/gpt-4.1" {
		t.Fatalf("expected model %q, got %q", "openai/gpt-4.1", fakeAdapter.streamReq.Model)
	}

	if err != nil {
		t.Fatalf("StreamChatCompletion returned err: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]

	if chunk.ID != "chatcmpl_mock" {
		t.Fatalf("expected ID %q, got %q", "chatcmpl_mock", chunk.ID)
	}

	if chunk.Object != "chat.completion.chunk" {
		t.Fatalf("expected object %q, got %q", "chat.completion.chunk", chunk.Object)
	}

	if chunk.Model != "openai/gpt-4.1" {
		t.Fatalf("expected model %q, got %q", "openai/gpt-4.1", chunk.Model)
	}

	if len(chunk.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(chunk.Choices))
	}

	if chunk.Choices[0].Index != 0 {
		t.Fatalf("expected index %d, got %d", 0, chunk.Choices[0].Index)
	}

	if chunk.Choices[0].Delta.Role != "assistant" {
		t.Fatalf("expected role %s, got %s", "assistant", chunk.Choices[0].Delta.Role)
	}

	if chunk.Choices[0].Delta.Content != "mock response" {
		t.Fatalf("expected content %s, got %s", "mock response", chunk.Choices[0].Delta.Content)
	}

	if chunk.Choices[0].FinishReason != nil {
		t.Fatalf("expected finish reason nil, got %q", *chunk.Choices[0].FinishReason)
	}
}
