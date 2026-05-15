package routing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

// fakeStore 是 routing 测试使用的候选 channel 存储替身。
type fakeStore struct {
	params sqlc.FindRouteCandidatesParams
	rows   []sqlc.FindRouteCandidatesRow
	err    error
}

// FindRouteCandidates 记录查询参数，并返回测试预设候选结果。
func (s *fakeStore) FindRouteCandidates(ctx context.Context, arg sqlc.FindRouteCandidatesParams) ([]sqlc.FindRouteCandidatesRow, error) {
	s.params = arg
	return s.rows, s.err
}

// fakeCredentialResolver 是 routing 测试使用的凭据解析器替身。
type fakeCredentialResolver struct {
	credentialRefs []string
	apiKey         string
	err            error
}

// Resolve 记录凭据引用，并返回测试预设 API key。
func (r *fakeCredentialResolver) Resolve(ctx context.Context, credentialRef string) (string, error) {
	r.credentialRefs = append(r.credentialRefs, credentialRef)
	return r.apiKey, r.err
}

func TestRouterPlanChatReturnsOrderedCandidates(t *testing.T) {
	store := &fakeStore{
		rows: []sqlc.FindRouteCandidatesRow{
			{
				RequestedModelID: "openai/gpt-4.1",
				ProviderID:       11,
				AdapterKey:       "openai",
				ChannelID:        123,
				BaseUrl:          "https://api.openai.example/v1",
				CredentialRef:    "secret://openai/main",
				TimeoutMs:        pgtype.Int4{Int32: 15000, Valid: true},
				UpstreamModel:    "gpt-4.1",
			},
			{
				RequestedModelID: "openai/gpt-4.1",
				ProviderID:       11,
				AdapterKey:       "openai",
				ChannelID:        456,
				BaseUrl:          "https://backup.openai.example/v1",
				CredentialRef:    "secret://openai/backup",
				TimeoutMs:        pgtype.Int4{Int32: 30000, Valid: true},
				UpstreamModel:    "gpt-4.1",
			},
		},
	}
	credentialResolver := &fakeCredentialResolver{apiKey: "resolved-secret"}
	router := NewRouter(store, credentialResolver, 30*time.Second)

	got, err := router.PlanChat(context.Background(), ChatRouteRequest{
		ProjectID: 42,
		ModelID:   "openai/gpt-4.1",
	})
	if err != nil {
		t.Fatalf("PlanChat returned error: %v", err)
	}

	if store.params.ProjectID != 42 {
		t.Fatalf("expected project id %d, got %d", int64(42), store.params.ProjectID)
	}
	if store.params.RequestedModelID != "openai/gpt-4.1" {
		t.Fatalf("expected requested model %q, got %q", "openai/gpt-4.1", store.params.RequestedModelID)
	}
	wantCredentialRefs := []string{"secret://openai/main", "secret://openai/backup"}
	if len(credentialResolver.credentialRefs) != len(wantCredentialRefs) {
		t.Fatalf("expected %d credential refs, got %#v", len(wantCredentialRefs), credentialResolver.credentialRefs)
	}
	for i, want := range wantCredentialRefs {
		if credentialResolver.credentialRefs[i] != want {
			t.Fatalf("expected credential ref %q at index %d, got %q", want, i, credentialResolver.credentialRefs[i])
		}
	}

	if got.RequestedModel != "openai/gpt-4.1" {
		t.Fatalf("expected requested model %q, got %q", "openai/gpt-4.1", got.RequestedModel)
	}
	if len(got.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got.Candidates))
	}

	first := got.Candidates[0]
	if first.ProviderID != 11 {
		t.Fatalf("expected provider id %d, got %d", int64(11), first.ProviderID)
	}
	if first.AdapterKey != "openai" {
		t.Fatalf("expected adapter key %q, got %q", "openai", first.AdapterKey)
	}
	if first.UpstreamModel != "gpt-4.1" {
		t.Fatalf("expected upstream model %q, got %q", "gpt-4.1", first.UpstreamModel)
	}
	if first.Channel.ID != 123 {
		t.Fatalf("expected channel id %d, got %d", int64(123), first.Channel.ID)
	}
	if first.Channel.BaseURL != "https://api.openai.example/v1" {
		t.Fatalf("expected base url %q, got %q", "https://api.openai.example/v1", first.Channel.BaseURL)
	}
	if first.Channel.APIKey != "resolved-secret" {
		t.Fatalf("expected resolved API key, got %q", first.Channel.APIKey)
	}
	if first.Channel.Timeout != 15*time.Second {
		t.Fatalf("expected timeout %v, got %v", 15*time.Second, first.Channel.Timeout)
	}

	second := got.Candidates[1]
	if second.Channel.ID != 456 {
		t.Fatalf("expected second channel id %d, got %d", int64(456), second.Channel.ID)
	}
	if second.Channel.Timeout != 30*time.Second {
		t.Fatalf("expected second timeout %v, got %v", 30*time.Second, second.Channel.Timeout)
	}
}

func TestNewRouterUsesFallbackDefaultTimeout(t *testing.T) {
	store := &fakeStore{
		rows: []sqlc.FindRouteCandidatesRow{
			{
				AdapterKey:    "openai",
				ChannelID:     123,
				BaseUrl:       "https://api.openai.example/v1",
				CredentialRef: "secret://openai/main",
				TimeoutMs:     pgtype.Int4{Valid: false},
				UpstreamModel: "gpt-4.1",
			},
		},
	}
	router := NewRouter(store, &fakeCredentialResolver{apiKey: "resolved-secret"}, 0)

	got, err := router.PlanChat(context.Background(), ChatRouteRequest{
		ProjectID: 42,
		ModelID:   "openai/gpt-4.1",
	})
	if err != nil {
		t.Fatalf("PlanChat returned error: %v", err)
	}

	if got.Candidates[0].Channel.Timeout != defaultChannelTimeout {
		t.Fatalf("expected fallback default timeout %v, got %v", defaultChannelTimeout, got.Candidates[0].Channel.Timeout)
	}
}

func TestRouterPlanChatReturnsNoAvailableChannel(t *testing.T) {
	router := NewRouter(&fakeStore{}, &fakeCredentialResolver{apiKey: "resolved-secret"}, 30*time.Second)

	_, err := router.PlanChat(context.Background(), ChatRouteRequest{
		ProjectID: 42,
		ModelID:   "openai/missing",
	})
	if !errors.Is(err, ErrNoAvailableChannel) {
		t.Fatalf("expected ErrNoAvailableChannel, got %v", err)
	}
}

func TestRouterPlanChatReturnsStoreError(t *testing.T) {
	storeErr := errors.New("database unavailable")
	router := NewRouter(&fakeStore{err: storeErr}, &fakeCredentialResolver{apiKey: "resolved-secret"}, 30*time.Second)

	_, err := router.PlanChat(context.Background(), ChatRouteRequest{
		ProjectID: 42,
		ModelID:   "openai/gpt-4.1",
	})
	if !errors.Is(err, storeErr) {
		t.Fatalf("expected store error, got %v", err)
	}
}

func TestRouterPlanChatReturnsCredentialResolverError(t *testing.T) {
	resolveErr := errors.New("secret manager unavailable")
	store := &fakeStore{
		rows: []sqlc.FindRouteCandidatesRow{
			{
				AdapterKey:    "openai",
				ChannelID:     123,
				BaseUrl:       "https://api.openai.example/v1",
				CredentialRef: "secret://openai/main",
				TimeoutMs:     pgtype.Int4{Int32: 15000, Valid: true},
				UpstreamModel: "gpt-4.1",
			},
		},
	}
	router := NewRouter(store, &fakeCredentialResolver{err: resolveErr}, 30*time.Second)

	_, err := router.PlanChat(context.Background(), ChatRouteRequest{
		ProjectID: 42,
		ModelID:   "openai/gpt-4.1",
	})
	if !errors.Is(err, resolveErr) {
		t.Fatalf("expected credential resolver error, got %v", err)
	}
}
