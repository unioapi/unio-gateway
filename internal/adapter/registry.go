package adapter

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidAdapterRegistration 表示 adapter 注册信息不合法。
	ErrInvalidAdapterRegistration = errors.New("invalid adapter registration")
	// ErrDuplicateAdapterKey 表示同一个 adapter key 被重复注册。
	ErrDuplicateAdapterKey = errors.New("duplicate adapter key")
)

// Registration 表示一个 adapter key 对应的代码能力。
type Registration struct {
	Key        string
	Chat       ChatAdapter
	StreamChat StreamChatAdapter
}

// Registry 根据 adapter key 查找对应 adapter 能力。
type Registry struct {
	chat       map[string]ChatAdapter
	streamChat map[string]StreamChatAdapter
}

// NewRegistry 创建 adapter registry。
func NewRegistry(registrations ...Registration) (*Registry, error) {
	r := &Registry{
		chat:       make(map[string]ChatAdapter),
		streamChat: make(map[string]StreamChatAdapter),
	}

	for _, reg := range registrations {
		if reg.Key == "" {
			return nil, fmt.Errorf("%w: empty key", ErrInvalidAdapterRegistration)
		}

		if reg.Chat == nil && reg.StreamChat == nil {
			return nil, fmt.Errorf("%w: %s has no capability", ErrInvalidAdapterRegistration, reg.Key)
		}

		if err := registerCapabilities(reg, r); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func registerCapabilities(reg Registration, r *Registry) error {
	if reg.Chat != nil {
		if _, exists := r.chat[reg.Key]; exists {
			return fmt.Errorf("%w: chat %s", ErrDuplicateAdapterKey, reg.Key)
		}
		r.chat[reg.Key] = reg.Chat
	}

	if reg.StreamChat != nil {
		if _, exists := r.streamChat[reg.Key]; exists {
			return fmt.Errorf("%w: stream %s", ErrDuplicateAdapterKey, reg.Key)
		}
		r.streamChat[reg.Key] = reg.StreamChat
	}

	return nil
}

// Chat 根据 adapter key 返回非流式聊天 adapter。
func (r *Registry) Chat(adapterKey string) (ChatAdapter, bool) {
	adapter, ok := r.chat[adapterKey]
	return adapter, ok
}

// StreamChat 根据 adapter key 返回流式聊天 adapter。
func (r *Registry) StreamChat(adapterKey string) (StreamChatAdapter, bool) {
	adapter, ok := r.streamChat[adapterKey]
	return adapter, ok
}
