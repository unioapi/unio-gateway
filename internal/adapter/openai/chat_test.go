package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/channel"
)

func TestAdapterChatCompletionsCallsUpstream(t *testing.T) {
	var gotAuthorization string
	var gotContentType string
	var gotRequestBody chatCompletionRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected method %q, got %q", http.MethodPost, r.Method)
		}

		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected path %q, got %q", "/v1/chat/completions", r.URL.Path)
		}

		gotAuthorization = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")

		if err := json.NewDecoder(r.Body).Decode(&gotRequestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(chatCompletionResponse{
			ID:    "chatcmpl_test",
			Model: "gpt-4.1",
			Choices: []chatChoice{
				{Message: chatMessage{Role: "assistant", Content: "hello from fake upstream"}},
			},
			Usage: chatCompletionUsage{
				PromptTokens:     11,
				CompletionTokens: 12,
				TotalTokens:      23,
			},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	openAIAdapter := NewAdapter(server.Client())

	selectedChannel := channel.Runtime{
		ID:      123,
		BaseURL: server.URL + "/v1",
		APIKey:  "test-secret",
		Timeout: 30 * time.Second,
	}

	got, err := openAIAdapter.ChatCompletions(context.Background(), selectedChannel, adapter.ChatRequest{
		Model: "gpt-4.1",
		Messages: []adapter.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAuthorization != "Bearer test-secret" {
		t.Fatalf("authorization header: got %q, want %q", gotAuthorization, "Bearer test-secret")
	}
	if !strings.HasPrefix(gotContentType, "application/json") {
		t.Fatalf("Content-type header does not contain application/json")
	}
	if gotRequestBody.Model != "gpt-4.1" {
		t.Fatalf("body model: got %q, want %q", gotRequestBody.Model, "gpt-4.1")
	}
	if len(gotRequestBody.Messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(gotRequestBody.Messages))
	}
	if gotRequestBody.Messages[0].Role != "user" {
		t.Fatalf("got role %q, want %q", gotRequestBody.Messages[0].Role, "user")
	}
	if gotRequestBody.Messages[0].Content != "hello" {
		t.Fatalf("got %q, want %q", gotRequestBody.Messages[0].Content, "hello")
	}

	if got.ID != "chatcmpl_test" {
		t.Fatalf("expected id %q, got %q", "chatcmpl_test", got.ID)
	}

	if got.Model != "gpt-4.1" {
		t.Fatalf("expected model %q, got %q", "gpt-4.1", got.Model)
	}

	if got.Content != "hello from fake upstream" {
		t.Fatalf("expected content %q, got %q", "hello from fake upstream", got.Content)
	}

	if got.Usage.PromptTokens != 11 {
		t.Fatalf("got prompt_tokens %d, want 11", got.Usage.PromptTokens)
	}
	if got.Usage.CompletionTokens != 12 {
		t.Fatalf("got completion_tokens %d, want 12", got.Usage.CompletionTokens)
	}
	if got.Usage.TotalTokens != 23 {
		t.Fatalf("got total_tokens %d, want 23", got.Usage.TotalTokens)
	}
}

func TestAdapterChatCompletionsReturnsErrorForUpstreamStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream failed", http.StatusBadGateway)
	}))
	defer server.Close()

	openAIAdapter := NewAdapter(server.Client())

	_, err := openAIAdapter.ChatCompletions(context.Background(), channel.Runtime{
		BaseURL: server.URL + "/v1",
		APIKey:  "test-secret",
		Timeout: 30 * time.Second,
	}, adapter.ChatRequest{
		Model: "gpt-4.1",
		Messages: []adapter.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "openai adapter: upstream status 502") {
		t.Fatalf("expected upstream status error, got %v", err)
	}
}

func TestAdapterChatCompletionsUsesChannelTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	openAIAdapter := NewAdapter(server.Client())

	_, err := openAIAdapter.ChatCompletions(
		context.Background(),
		channel.Runtime{
			BaseURL: server.URL + "/v1",
			APIKey:  "test-secret",
			Timeout: 50 * time.Millisecond,
		},
		adapter.ChatRequest{Model: "gpt-4.1",
			Messages: []adapter.ChatMessage{
				{Role: "user", Content: "hello"},
			}})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}
