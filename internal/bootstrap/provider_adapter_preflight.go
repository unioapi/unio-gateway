package bootstrap

import (
	"context"
	"errors"

	"github.com/ThankCat/unio-api/internal/platform/failure"
	"github.com/ThankCat/unio-api/internal/platform/store/sqlc"
)

var (
	// ErrProviderAdapterCapabilityMissing 表示启用 provider 配置的 adapter 缺少当前进程要求的能力。
	ErrProviderAdapterCapabilityMissing = errors.New("provider adapter capability missing")
)

// ProviderAdapterStore 定义 provider adapter preflight 需要的最小存储能力。
type ProviderAdapterStore interface {
	ListEnabledProviderAdapters(ctx context.Context) ([]sqlc.ListEnabledProviderAdaptersRow, error)
}

// AdapterCapabilityRegistry 定义 provider adapter preflight 需要查询的 adapter 能力注册表。
type AdapterCapabilityRegistry interface {
	HasChat(adapterKey string) bool
	HasStreamChat(adapterKey string) bool
}

// ProviderAdapterPreflight 校验启用 provider 的 adapter 配置是否能被当前进程支持。
type ProviderAdapterPreflight struct {
	store    ProviderAdapterStore
	registry AdapterCapabilityRegistry
}

// NewProviderAdapterPreflight 创建 provider adapter 启动前校验器。
func NewProviderAdapterPreflight(store ProviderAdapterStore, registry AdapterCapabilityRegistry) *ProviderAdapterPreflight {
	return &ProviderAdapterPreflight{
		store:    store,
		registry: registry,
	}
}

// ValidateChatCapabilities 校验所有启用 provider 的 adapter 都支持 chat 和 stream chat。
func (p *ProviderAdapterPreflight) ValidateChatCapabilities(ctx context.Context) error {
	rows, err := p.store.ListEnabledProviderAdapters(ctx)
	if err != nil {
		return failure.Wrap(
			failure.CodeBootstrapStoreFailed,
			err,
			failure.WithMessage("failed to list enabled provider adapters"),
		)
	}

	for _, row := range rows {
		if !p.registry.HasChat(row.Adapter) {
			return failure.Wrap(
				failure.CodeBootstrapProviderAdapterCapabilityMissing,
				ErrProviderAdapterCapabilityMissing,
				failure.WithMessage("provider adapter chat capability missing"),
				failure.WithField("provider_id", row.ID),
				failure.WithField("provider_slug", row.Slug),
				failure.WithField("adapter_key", row.Adapter),
				failure.WithField("capability", "chat"),
			)
		}

		if !p.registry.HasStreamChat(row.Adapter) {
			return failure.Wrap(
				failure.CodeBootstrapProviderAdapterCapabilityMissing,
				ErrProviderAdapterCapabilityMissing,
				failure.WithMessage("provider adapter stream chat capability missing"),
				failure.WithField("provider_id", row.ID),
				failure.WithField("provider_slug", row.Slug),
				failure.WithField("adapter_key", row.Adapter),
				failure.WithField("capability", "stream_chat"),
			)
		}
	}

	return nil
}
