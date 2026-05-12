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
	called bool
	ch     channel.Runtime
	req    adapter.ChatRequest
	resp   *adapter.ChatResponse
	err    error
}

// ChatCompletions 记录 gateway 传入的请求，并返回测试预设响应。
func (a *fakeChatAdapter) ChatCompletions(ctx context.Context, ch channel.Runtime, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
	a.called = true
	a.ch = ch
	a.req = req

	return a.resp, a.err
}

func TestChatCompletionServiceCreateChatCompletionCallsAdapter(t *testing.T) {
	selectedChannel := channel.Runtime{
		ID:      123,
		BaseURL: "https://example.test/v1",
		APIKey:  "test-secret",
		Timeout: 30 * time.Second,
	}
	fakeAdapter := &fakeChatAdapter{
		resp: &adapter.ChatResponse{
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

	if !fakeAdapter.called {
		t.Fatal("expected adapter to be called")
	}

	if fakeAdapter.req.Model != "openai/gpt-4.1" {
		t.Fatalf("expected model %q, got %q", "openai/gpt-4.1", fakeAdapter.req.Model)
	}

	if len(fakeAdapter.req.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(fakeAdapter.req.Messages))
	}

	if len(got.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(got.Choices))
	}

	if fakeAdapter.req.Messages[0].Content != "hello" {
		t.Fatalf("expected message content %q, got %q", "hello", fakeAdapter.req.Messages[0].Content)
	}

	if fakeAdapter.req.Messages[0].Role != "user" {
		t.Fatalf("expected message role %q, got %q", "user", fakeAdapter.req.Messages[0].Role)
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
