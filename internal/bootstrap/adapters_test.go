package bootstrap

import "testing"

func TestNewAdapterRegistryRegistersOpenAIChatCapabilities(t *testing.T) {
	registry, err := NewAdapterRegistry(nil)
	if err != nil {
		t.Fatalf("NewAdapterRegistry returned error: %v", err)
	}

	if !registry.HasChat("openai") {
		t.Fatal("expected openai chat capability to be registered")
	}
	if !registry.HasStreamChat("openai") {
		t.Fatal("expected openai stream chat capability to be registered")
	}
	if registry.HasChat("missing") {
		t.Fatal("expected unknown chat capability to be absent")
	}
	if registry.HasStreamChat("missing") {
		t.Fatal("expected unknown stream chat capability to be absent")
	}
}
