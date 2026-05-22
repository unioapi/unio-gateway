package adapter

import (
	"context"
	"errors"
	"testing"

	"github.com/ThankCat/unio-api/internal/channel"
)

// registryTestChatAdapter 是 registry 测试使用的非流式 adapter 替身。
type registryTestChatAdapter struct{}

// ChatCompletions 实现 ChatAdapter，registry 测试不关心实际调用结果。
func (a *registryTestChatAdapter) ChatCompletions(ctx context.Context, ch channel.Runtime, req ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{}, nil
}

// registryTestStreamChatAdapter 是 registry 测试使用的流式 adapter 替身。
type registryTestStreamChatAdapter struct{}

// StreamChatCompletions 实现 StreamChatAdapter，registry 测试不关心实际流式内容。
func (a *registryTestStreamChatAdapter) StreamChatCompletions(ctx context.Context, ch channel.Runtime, req ChatRequest, emit func(ChatStreamChunk) error) error {
	return nil
}

func TestRegistryReturnsRegisteredChatAdapter(t *testing.T) {
	chatAdapter := &registryTestChatAdapter{}
	registry, err := NewRegistry(Registration{
		Key:  "openai",
		Chat: chatAdapter,
	})
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	got, ok := registry.Chat("openai")
	if !ok {
		t.Fatal("expected registered chat adapter")
	}
	if got != chatAdapter {
		t.Fatal("expected registered chat adapter instance")
	}

	if _, ok := registry.StreamChat("openai"); ok {
		t.Fatal("expected missing stream chat adapter")
	}
}

func TestRegistryReturnsRegisteredStreamChatAdapter(t *testing.T) {
	streamAdapter := &registryTestStreamChatAdapter{}
	registry, err := NewRegistry(Registration{
		Key:        "openai",
		StreamChat: streamAdapter,
	})
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	got, ok := registry.StreamChat("openai")
	if !ok {
		t.Fatal("expected registered stream chat adapter")
	}
	if got != streamAdapter {
		t.Fatal("expected registered stream chat adapter instance")
	}

	if _, ok := registry.Chat("openai"); ok {
		t.Fatal("expected missing chat adapter")
	}
}

func TestRegistryReturnsFalseForUnknownAdapterKey(t *testing.T) {
	registry, err := NewRegistry(Registration{
		Key:  "openai",
		Chat: &registryTestChatAdapter{},
	})
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	if _, ok := registry.Chat("anthropic"); ok {
		t.Fatal("expected unknown chat adapter key to return false")
	}
	if _, ok := registry.StreamChat("anthropic"); ok {
		t.Fatal("expected unknown stream chat adapter key to return false")
	}
}

func TestRegistryReportsRegisteredCapabilities(t *testing.T) {
	registry, err := NewRegistry(
		Registration{
			Key:        "openai",
			Chat:       &registryTestChatAdapter{},
			StreamChat: &registryTestStreamChatAdapter{},
		},
		Registration{
			Key:  "chat-only",
			Chat: &registryTestChatAdapter{},
		},
		Registration{
			Key:        "stream-only",
			StreamChat: &registryTestStreamChatAdapter{},
		},
	)
	if err != nil {
		t.Fatalf("NewRegistry returned error: %v", err)
	}

	tests := []struct {
		name       string
		key        string
		wantChat   bool
		wantStream bool
	}{
		{name: "both capabilities", key: "openai", wantChat: true, wantStream: true},
		{name: "chat only", key: "chat-only", wantChat: true, wantStream: false},
		{name: "stream only", key: "stream-only", wantChat: false, wantStream: true},
		{name: "unknown key", key: "missing", wantChat: false, wantStream: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := registry.HasChat(tt.key); got != tt.wantChat {
				t.Fatalf("HasChat(%q) = %v, want %v", tt.key, got, tt.wantChat)
			}

			if got := registry.HasStreamChat(tt.key); got != tt.wantStream {
				t.Fatalf("HasStreamChat(%q) = %v, want %v", tt.key, got, tt.wantStream)
			}
		})
	}
}

func TestNewRegistryRejectsEmptyKey(t *testing.T) {
	_, err := NewRegistry(Registration{
		Key:  "",
		Chat: &registryTestChatAdapter{},
	})
	if !errors.Is(err, ErrInvalidAdapterRegistration) {
		t.Fatalf("expected ErrInvalidAdapterRegistration, got %v", err)
	}
}

func TestNewRegistryRejectsMissingCapabilities(t *testing.T) {
	_, err := NewRegistry(Registration{
		Key: "openai",
	})
	if !errors.Is(err, ErrInvalidAdapterRegistration) {
		t.Fatalf("expected ErrInvalidAdapterRegistration, got %v", err)
	}
}

func TestNewRegistryRejectsDuplicateChatAdapterKey(t *testing.T) {
	_, err := NewRegistry(
		Registration{
			Key:  "openai",
			Chat: &registryTestChatAdapter{},
		},
		Registration{
			Key:  "openai",
			Chat: &registryTestChatAdapter{},
		},
	)
	if !errors.Is(err, ErrDuplicateAdapterKey) {
		t.Fatalf("expected ErrDuplicateAdapterKey, got %v", err)
	}
}

func TestNewRegistryRejectsDuplicateStreamChatAdapterKey(t *testing.T) {
	_, err := NewRegistry(
		Registration{
			Key:        "openai",
			StreamChat: &registryTestStreamChatAdapter{},
		},
		Registration{
			Key:        "openai",
			StreamChat: &registryTestStreamChatAdapter{},
		},
	)
	if !errors.Is(err, ErrDuplicateAdapterKey) {
		t.Fatalf("expected ErrDuplicateAdapterKey, got %v", err)
	}
}
