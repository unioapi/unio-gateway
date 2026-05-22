package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/ThankCat/unio-api/internal/credential"
	"github.com/ThankCat/unio-api/internal/failure"
	"github.com/ThankCat/unio-api/internal/routing"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeChatRouteStore struct {
	rows []sqlc.FindRouteCandidatesRow
}

func (s *fakeChatRouteStore) ModelExistsByID(ctx context.Context, requestedModelID string) (bool, error) {
	return true, nil
}

func (s *fakeChatRouteStore) ProjectCanUseModel(ctx context.Context, arg sqlc.ProjectCanUseModelParams) (bool, error) {
	return true, nil
}

func (s *fakeChatRouteStore) FindRouteCandidates(ctx context.Context, arg sqlc.FindRouteCandidatesParams) ([]sqlc.FindRouteCandidatesRow, error) {
	return s.rows, nil
}

func TestNewChatRouterCreatesRouterWithStaticCredentialResolver(t *testing.T) {
	router, err := NewChatRouter(&fakeChatRouteStore{
		rows: []sqlc.FindRouteCandidatesRow{
			{
				ModelDbID:     7,
				ProviderID:    11,
				AdapterKey:    "openai",
				ChannelID:     13,
				BaseUrl:       "https://api.openai.example/v1",
				CredentialRef: "secret://missing",
				TimeoutMs:     pgtype.Int4{Int32: 15000, Valid: true},
				UpstreamModel: "gpt-4.1",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewChatRouter returned error: %v", err)
	}
	if router == nil {
		t.Fatal("expected chat router")
	}

	_, err = router.PlanChat(context.Background(), routing.ChatRouteRequest{
		ProjectID: 1,
		ModelID:   "gpt-4.1",
	})
	if !errors.Is(err, credential.ErrCredentialNotFound) {
		t.Fatalf("expected static resolver credential not found error, got %v", err)
	}
	if got := failure.CodeOf(err); got != failure.CodeRoutingCredentialResolveFailed {
		t.Fatalf("expected code %q, got %q", failure.CodeRoutingCredentialResolveFailed, got)
	}
}
