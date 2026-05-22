package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/ThankCat/unio-api/internal/failure"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
)

type fakeProviderAdapterStore struct {
	rows []sqlc.ListEnabledProviderAdaptersRow
	err  error
}

func (s *fakeProviderAdapterStore) ListEnabledProviderAdapters(ctx context.Context) ([]sqlc.ListEnabledProviderAdaptersRow, error) {
	return s.rows, s.err
}

type fakeAdapterCapabilityRegistry struct {
	chat       map[string]bool
	streamChat map[string]bool
}

func (r *fakeAdapterCapabilityRegistry) HasChat(adapterKey string) bool {
	return r.chat[adapterKey]
}

func (r *fakeAdapterCapabilityRegistry) HasStreamChat(adapterKey string) bool {
	return r.streamChat[adapterKey]
}

func TestProviderAdapterPreflightAcceptsRegisteredChatCapabilities(t *testing.T) {
	preflight := NewProviderAdapterPreflight(
		&fakeProviderAdapterStore{
			rows: []sqlc.ListEnabledProviderAdaptersRow{
				{ID: 1, Slug: "openai", Adapter: "openai"},
				{ID: 2, Slug: "deepseek", Adapter: "deepseek"},
			},
		},
		&fakeAdapterCapabilityRegistry{
			chat: map[string]bool{
				"openai":   true,
				"deepseek": true,
			},
			streamChat: map[string]bool{
				"openai":   true,
				"deepseek": true,
			},
		},
	)

	if err := preflight.ValidateChatCapabilities(context.Background()); err != nil {
		t.Fatalf("ValidateChatCapabilities returned error: %v", err)
	}
}

func TestProviderAdapterPreflightAcceptsNoEnabledProviders(t *testing.T) {
	preflight := NewProviderAdapterPreflight(
		&fakeProviderAdapterStore{},
		&fakeAdapterCapabilityRegistry{},
	)

	if err := preflight.ValidateChatCapabilities(context.Background()); err != nil {
		t.Fatalf("ValidateChatCapabilities returned error: %v", err)
	}
}

func TestProviderAdapterPreflightWrapsStoreError(t *testing.T) {
	storeErr := errors.New("database unavailable")
	preflight := NewProviderAdapterPreflight(
		&fakeProviderAdapterStore{err: storeErr},
		&fakeAdapterCapabilityRegistry{},
	)

	err := preflight.ValidateChatCapabilities(context.Background())
	if !errors.Is(err, storeErr) {
		t.Fatalf("expected store error cause, got %v", err)
	}
	if got := failure.CodeOf(err); got != failure.CodeBootstrapStoreFailed {
		t.Fatalf("expected code %q, got %q", failure.CodeBootstrapStoreFailed, got)
	}
	if got := failure.CategoryOf(err); got != failure.CategoryBootstrap {
		t.Fatalf("expected category %q, got %q", failure.CategoryBootstrap, got)
	}
}

func TestProviderAdapterPreflightRejectsMissingChatCapability(t *testing.T) {
	preflight := NewProviderAdapterPreflight(
		&fakeProviderAdapterStore{
			rows: []sqlc.ListEnabledProviderAdaptersRow{
				{ID: 42, Slug: "openai", Adapter: "openai"},
			},
		},
		&fakeAdapterCapabilityRegistry{
			streamChat: map[string]bool{"openai": true},
		},
	)

	err := preflight.ValidateChatCapabilities(context.Background())
	assertCapabilityMissingError(t, err, map[string]any{
		"provider_id":   int64(42),
		"provider_slug": "openai",
		"adapter_key":   "openai",
		"capability":    "chat",
	})
}

func TestProviderAdapterPreflightRejectsMissingStreamChatCapability(t *testing.T) {
	preflight := NewProviderAdapterPreflight(
		&fakeProviderAdapterStore{
			rows: []sqlc.ListEnabledProviderAdaptersRow{
				{ID: 99, Slug: "deepseek", Adapter: "deepseek"},
			},
		},
		&fakeAdapterCapabilityRegistry{
			chat: map[string]bool{"deepseek": true},
		},
	)

	err := preflight.ValidateChatCapabilities(context.Background())
	assertCapabilityMissingError(t, err, map[string]any{
		"provider_id":   int64(99),
		"provider_slug": "deepseek",
		"adapter_key":   "deepseek",
		"capability":    "stream_chat",
	})
}

func assertCapabilityMissingError(t *testing.T, err error, wantFields map[string]any) {
	t.Helper()

	if !errors.Is(err, ErrProviderAdapterCapabilityMissing) {
		t.Fatalf("expected ErrProviderAdapterCapabilityMissing, got %v", err)
	}
	if got := failure.CodeOf(err); got != failure.CodeBootstrapProviderAdapterCapabilityMissing {
		t.Fatalf("expected code %q, got %q", failure.CodeBootstrapProviderAdapterCapabilityMissing, got)
	}
	if got := failure.CategoryOf(err); got != failure.CategoryBootstrap {
		t.Fatalf("expected category %q, got %q", failure.CategoryBootstrap, got)
	}

	gotFields := map[string]any{}
	for _, field := range failure.FieldsOf(err) {
		gotFields[field.Key] = field.Value
	}

	for key, want := range wantFields {
		if gotFields[key] != want {
			t.Fatalf("expected field %s=%v, got %v in %#v", key, want, gotFields[key], gotFields)
		}
	}
}
