package sqlc_test

import (
	"testing"
	"time"
)

func TestListEnabledProviderAdaptersFiltersDisabledAndOrdersBySlug(t *testing.T) {
	ctx, tx, queries, cleanup := newModelChannelTestTx(t)
	defer cleanup()

	suffix := time.Now().UnixNano()
	alphaSlug := "provider-adapter-alpha-" + time.Unix(0, suffix).Format("20060102150405.000000000")
	betaSlug := "provider-adapter-beta-" + time.Unix(0, suffix).Format("20060102150405.000000000")
	disabledSlug := "provider-adapter-disabled-" + time.Unix(0, suffix).Format("20060102150405.000000000")

	betaID := insertProvider(t, ctx, tx, betaSlug, "anthropic", "enabled")
	disabledID := insertProvider(t, ctx, tx, disabledSlug, "gemini", "disabled")
	alphaID := insertProvider(t, ctx, tx, alphaSlug, "openai", "enabled")

	rows, err := queries.ListEnabledProviderAdapters(ctx)
	if err != nil {
		t.Fatalf("list enabled provider adapters: %v", err)
	}

	type providerAdapter struct {
		id      int64
		slug    string
		adapter string
	}

	var got []providerAdapter
	for _, row := range rows {
		switch row.Slug {
		case alphaSlug, betaSlug:
			got = append(got, providerAdapter{
				id:      row.ID,
				slug:    row.Slug,
				adapter: row.Adapter,
			})
		case disabledSlug:
			t.Fatalf("disabled provider %q with id %d should not be returned", row.Slug, disabledID)
		}
	}

	want := []providerAdapter{
		{id: alphaID, slug: alphaSlug, adapter: "openai"},
		{id: betaID, slug: betaSlug, adapter: "anthropic"},
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d test providers, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("provider %d: got %#v, want %#v", i, got[i], want[i])
		}
	}
}
