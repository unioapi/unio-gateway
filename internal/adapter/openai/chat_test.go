package openai

import (
	"context"
	"encoding/json"
	"errors"
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

func TestAdapterStreamChatCompletionsParsesUpstreamSSE(t *testing.T) {
	var gotAuthorization string
	var gotAccept string
	var gotContentType string
	var gotRequestBody chatCompletionRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("got %q, want %q", r.Method, http.MethodPost)
		}

		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("got %q, want %q", r.URL.Path, "/v1/chat/completions")
		}

		gotAuthorization = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")

		if err := json.NewDecoder(r.Body).Decode(&gotRequestBody); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}

		if !gotRequestBody.Stream {
			t.Fatal("expected stream request body to be true")
		}

		w.Header().Set("Content-Type", "text/event-stream")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected response writer to support flush")
		}
		stop := "stop"
		chunks := []chatCompletionStreamResponse{
			{
				ID:    "chatcmpl_stream_test",
				Model: "gpt-4.1",
				Choices: []chatStreamChoice{
					{
						Delta:        chatStreamDelta{},
						FinishReason: nil,
					},
				},
			},
			{
				ID:    "chatcmpl_stream_test",
				Model: "gpt-4.1",
				Choices: []chatStreamChoice{
					{
						Delta: chatStreamDelta{
							Role:    "assistant",
							Content: "hello ",
						},
						FinishReason: nil,
					},
				},
			},
			{
				ID:    "chatcmpl_stream_test",
				Model: "gpt-4.1",
				Choices: []chatStreamChoice{
					{
						Delta: chatStreamDelta{
							Content: "world",
						},
						FinishReason: &stop,
					},
				},
			},
		}

		for _, chunk := range chunks {
			payload, err := json.Marshal(chunk)
			if err != nil {
				t.Fatalf("failed to marshal chunk: %v", err)
			}

			if _, err := w.Write([]byte("data: " + string(payload) + "\n\n")); err != nil {
				t.Fatalf("write stream chunk: %v", err)
			}
			flusher.Flush()
		}

		if _, err := w.Write([]byte("data: [DONE]\n\n")); err != nil {
			t.Fatalf("write done chunk: %v", err)
		}
		flusher.Flush()
	}))
	defer server.Close()

	openAIAdapter := NewAdapter(server.Client())
	selectedChannel := channel.Runtime{
		ID:      123,
		BaseURL: server.URL + "/v1",
		APIKey:  "test-secret",
		Timeout: 30 * time.Second,
	}

	got := make([]adapter.ChatStreamChunk, 0)
	err := openAIAdapter.StreamChatCompletions(context.Background(), selectedChannel, adapter.ChatRequest{
		Model: "gpt-4.1",
		Messages: []adapter.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}, func(chunk adapter.ChatStreamChunk) error {
		got = append(got, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamChatCompletions returned err: %v", err)
	}

	if gotAuthorization != "Bearer test-secret" {
		t.Fatalf("authorization header: got %q, want %q", gotAuthorization, "Bearer test-secret")
	}

	if gotAccept != "text/event-stream" {
		t.Fatalf("accept header: got %q, want %q", gotAccept, "text/event-stream")
	}

	if !strings.HasPrefix(gotContentType, "application/json") {
		t.Fatalf("content type: got %q, want application/json", gotContentType)
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
		t.Fatalf("got content %q, want %q", gotRequestBody.Messages[0].Content, "hello")
	}

	if len(got) != 2 {
		t.Fatalf("got %d chunks, want 2", len(got))
	}

	if got[0].ID != "chatcmpl_stream_test" {
		t.Fatalf("got id %q, want %q", got[0].ID, "chatcmpl_stream_test")
	}
	if got[0].Model != "gpt-4.1" {
		t.Fatalf("got model %q, want %q", got[0].Model, "gpt-4.1")
	}
	if got[0].Role != "assistant" {
		t.Fatalf("got role %q, want %q", got[0].Role, "assistant")
	}
	if got[0].Content != "hello " {
		t.Fatalf("got content %q, want %q", got[0].Content, "hello ")
	}

	if got[1].Content != "world" {
		t.Fatalf("got content %q, want %q", got[1].Content, "world")
	}
	if got[1].FinishReason == nil {
		t.Fatal("got nil finish reason, want stop")
	}
	if *got[1].FinishReason != "stop" {
		t.Fatalf("got finish reason %q, want %q", *got[1].FinishReason, "stop")
	}
}

func TestAdapterStreamChatCompletionsReturnsErrorForUpstreamStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream stream failed", http.StatusBadGateway)
	}))
	defer server.Close()

	openAIAdapter := NewAdapter(server.Client())

	err := openAIAdapter.StreamChatCompletions(context.Background(), channel.Runtime{
		BaseURL: server.URL + "/v1",
		APIKey:  "test-secret",
		Timeout: 30 * time.Second,
	}, adapter.ChatRequest{
		Model: "gpt-4.1",
		Messages: []adapter.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}, func(chunk adapter.ChatStreamChunk) error {
		t.Fatalf("unexpected stream chunk: %+v", chunk)
		return nil
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "openai adapter: upstream stream status 502") {
		t.Fatalf("expected upstream stream status error, got %v", err)
	}
}

func TestAdapterStreamChatCompletionsReturnsErrorForBadSSEJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if _, err := w.Write([]byte("data: {bad json}\n\n")); err != nil {
			t.Fatalf("write stream chunk: %v", err)
		}
	}))
	defer server.Close()

	openAIAdapter := NewAdapter(server.Client())

	err := openAIAdapter.StreamChatCompletions(context.Background(), channel.Runtime{
		BaseURL: server.URL + "/v1",
		APIKey:  "test-secret",
		Timeout: 30 * time.Second,
	}, adapter.ChatRequest{
		Model: "gpt-4.1",
		Messages: []adapter.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}, func(chunk adapter.ChatStreamChunk) error {
		t.Fatalf("unexpected stream chunk: %+v", chunk)
		return nil
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "openai adapter: decode stream chunk") {
		t.Fatalf("expected decode stream chunk error, got %v", err)
	}
}

func TestAdapterStreamChatCompletionsStopsWhenEmitReturnsError(t *testing.T) {
	emitErr := errors.New("emit failed")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		chunks := []chatCompletionStreamResponse{
			{
				ID:    "chatcmpl_stream_test",
				Model: "gpt-4.1",
				Choices: []chatStreamChoice{
					{Delta: chatStreamDelta{Content: "first"}},
				},
			},
			{
				ID:    "chatcmpl_stream_test",
				Model: "gpt-4.1",
				Choices: []chatStreamChoice{
					{Delta: chatStreamDelta{Content: "second"}},
				},
			},
		}

		for _, chunk := range chunks {
			payload, err := json.Marshal(chunk)
			if err != nil {
				t.Fatalf("failed to marshal chunk: %v", err)
			}

			if _, err := w.Write([]byte("data: " + string(payload) + "\n\n")); err != nil {
				t.Fatalf("write stream chunk: %v", err)
			}
		}
	}))
	defer server.Close()

	openAIAdapter := NewAdapter(server.Client())

	emitCalls := 0
	err := openAIAdapter.StreamChatCompletions(context.Background(), channel.Runtime{
		BaseURL: server.URL + "/v1",
		APIKey:  "test-secret",
		Timeout: 30 * time.Second,
	}, adapter.ChatRequest{
		Model: "gpt-4.1",
		Messages: []adapter.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}, func(chunk adapter.ChatStreamChunk) error {
		emitCalls++
		return emitErr
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, emitErr) {
		t.Fatalf("expected emit error wrapping %v, got %v", emitErr, err)
	}

	if !strings.Contains(err.Error(), "openai adapter: send stream chunk") {
		t.Fatalf("expected send stream chunk error, got %v", err)
	}

	if emitCalls != 1 {
		t.Fatalf("got %d emit calls, want 1", emitCalls)
	}
}

func TestAdapterStreamChatCompletionsStopsAtDone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		firstChunk := chatCompletionStreamResponse{
			ID:    "chatcmpl_stream_test",
			Model: "gpt-4.1",
			Choices: []chatStreamChoice{
				{Delta: chatStreamDelta{Content: "before done"}},
			},
		}
		payload, err := json.Marshal(firstChunk)
		if err != nil {
			t.Fatalf("failed to marshal chunk: %v", err)
		}

		if _, err := w.Write([]byte("data: " + string(payload) + "\n\n")); err != nil {
			t.Fatalf("write first chunk: %v", err)
		}

		if _, err := w.Write([]byte("data: [DONE]\n\n")); err != nil {
			t.Fatalf("write done chunk: %v", err)
		}

		afterDoneChunk := chatCompletionStreamResponse{
			ID:    "chatcmpl_stream_test",
			Model: "gpt-4.1",
			Choices: []chatStreamChoice{
				{Delta: chatStreamDelta{Content: "after done"}},
			},
		}
		payload, err = json.Marshal(afterDoneChunk)
		if err != nil {
			t.Fatalf("failed to marshal chunk after done: %v", err)
		}

		// 这段模拟异常上游在 [DONE] 后继续输出，adapter 应该忽略后续内容。
		if _, err := w.Write([]byte("data: " + string(payload) + "\n\n")); err != nil {
			t.Fatalf("write chunk after done: %v", err)
		}
	}))
	defer server.Close()

	openAIAdapter := NewAdapter(server.Client())

	got := make([]adapter.ChatStreamChunk, 0)
	err := openAIAdapter.StreamChatCompletions(context.Background(), channel.Runtime{
		BaseURL: server.URL + "/v1",
		APIKey:  "test-secret",
		Timeout: 30 * time.Second,
	}, adapter.ChatRequest{
		Model: "gpt-4.1",
		Messages: []adapter.ChatMessage{
			{Role: "user", Content: "hello"},
		},
	}, func(chunk adapter.ChatStreamChunk) error {
		got = append(got, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d chunks, want 1", len(got))
	}

	if got[0].Content != "before done" {
		t.Fatalf("got content %q, want %q", got[0].Content, "before done")
	}
}
