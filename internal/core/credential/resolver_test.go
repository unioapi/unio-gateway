package credential

import (
	"context"
	"testing"
)

func TestStaticResolverResolve(t *testing.T) {
	resolver := NewStaticResolver(map[string]string{
		"secret://openai/main": "sk-test",
	})

	got, err := resolver.Resolve(context.Background(), "secret://openai/main")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got != "sk-test" {
		t.Fatalf("expected credential %q, got %q", "sk-test", got)
	}
}

func TestStaticResolverCopiesInputValues(t *testing.T) {
	values := map[string]string{
		"secret://openai/main": "sk-original",
	}
	resolver := NewStaticResolver(values)
	values["secret://openai/main"] = "sk-mutated"

	got, err := resolver.Resolve(context.Background(), "secret://openai/main")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got != "sk-original" {
		t.Fatalf("expected copied credential %q, got %q", "sk-original", got)
	}
}

func TestStaticResolverRejectsEmptyCredentialRef(t *testing.T) {
	resolver := NewStaticResolver(map[string]string{})

	if _, err := resolver.Resolve(context.Background(), "   "); err == nil {
		t.Fatal("expected empty credential ref error")
	}
}

func TestStaticResolverRejectsUnknownCredentialRef(t *testing.T) {
	resolver := NewStaticResolver(map[string]string{})

	if _, err := resolver.Resolve(context.Background(), "secret://missing"); err == nil {
		t.Fatal("expected unknown credential ref error")
	}
}
