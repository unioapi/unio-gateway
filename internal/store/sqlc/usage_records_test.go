package sqlc_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5/pgconn"
)

// isCheckViolation 判断数据库错误是否是 CHECK 约束冲突。
func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

func TestUsageRecordCreateAndGetByRequest(t *testing.T) {
	ctx, _, queries, cleanup := newModelChannelTestTx(t)
	defer cleanup()

	identity := createRequestRecordIdentity(t, ctx, queries)
	requestID := fmt.Sprintf("usage-record-%d", time.Now().UnixNano())
	requestRecord := createRequestRecordForTest(t, ctx, queries, identity, requestID)

	created, err := queries.CreateUsageRecord(ctx, sqlc.CreateUsageRecordParams{
		RequestRecordID:  requestRecord.ID,
		PromptTokens:     100,
		CompletionTokens: 40,
		TotalTokens:      140,
		CachedTokens:     12,
		ReasoningTokens:  8,
		Source:           "upstream_response",
	})
	if err != nil {
		t.Fatalf("create usage record: %v", err)
	}

	if created.ID == 0 {
		t.Fatal("expected usage record id")
	}
	if created.RequestRecordID != requestRecord.ID {
		t.Fatalf("expected request_record_id %d, got %d", requestRecord.ID, created.RequestRecordID)
	}
	if created.PromptTokens != 100 || created.CompletionTokens != 40 || created.TotalTokens != 140 {
		t.Fatalf("expected token usage 100/40/140, got %d/%d/%d", created.PromptTokens, created.CompletionTokens, created.TotalTokens)
	}
	if created.CachedTokens != 12 || created.ReasoningTokens != 8 {
		t.Fatalf("expected cached/reasoning usage 12/8, got %d/%d", created.CachedTokens, created.ReasoningTokens)
	}
	if created.Source != "upstream_response" {
		t.Fatalf("expected source upstream_response, got %q", created.Source)
	}
	if !created.CreatedAt.Valid {
		t.Fatal("expected created_at to be set")
	}

	got, err := queries.GetUsageRecordByRequest(ctx, requestRecord.ID)
	if err != nil {
		t.Fatalf("get usage by request: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected usage id %d, got %d", created.ID, got.ID)
	}
	if got.TotalTokens != 140 {
		t.Fatalf("expected total tokens 140, got %d", got.TotalTokens)
	}
}

func TestUsageRecordRejectsDuplicateRequest(t *testing.T) {
	ctx, _, queries, cleanup := newModelChannelTestTx(t)
	defer cleanup()

	identity := createRequestRecordIdentity(t, ctx, queries)
	requestID := fmt.Sprintf("usage-duplicate-%d", time.Now().UnixNano())
	requestRecord := createRequestRecordForTest(t, ctx, queries, identity, requestID)

	params := sqlc.CreateUsageRecordParams{
		RequestRecordID:  requestRecord.ID,
		PromptTokens:     10,
		CompletionTokens: 5,
		TotalTokens:      15,
		CachedTokens:     0,
		ReasoningTokens:  0,
		Source:           "upstream_response",
	}

	if _, err := queries.CreateUsageRecord(ctx, params); err != nil {
		t.Fatalf("create first usage record: %v", err)
	}

	_, err := queries.CreateUsageRecord(ctx, params)
	if err == nil {
		t.Fatal("expected duplicate request_record_id error")
	}
	if !isUniqueViolation(err) {
		t.Fatalf("expected unique violation, got %v", err)
	}
}

func TestUsageRecordRejectsInvalidTokenConstraints(t *testing.T) {
	ctx, _, queries, cleanup := newModelChannelTestTx(t)
	defer cleanup()

	identity := createRequestRecordIdentity(t, ctx, queries)

	cases := []struct {
		name   string
		params sqlc.CreateUsageRecordParams
	}{
		{
			name: "negative prompt tokens",
			params: sqlc.CreateUsageRecordParams{
				PromptTokens:     -1,
				CompletionTokens: 5,
				TotalTokens:      4,
				CachedTokens:     0,
				ReasoningTokens:  0,
				Source:           "upstream_response",
			},
		},
		{
			name: "total token mismatch",
			params: sqlc.CreateUsageRecordParams{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      16,
				CachedTokens:     0,
				ReasoningTokens:  0,
				Source:           "upstream_response",
			},
		},
		{
			name: "cached tokens exceed prompt tokens",
			params: sqlc.CreateUsageRecordParams{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
				CachedTokens:     11,
				ReasoningTokens:  0,
				Source:           "upstream_response",
			},
		},
		{
			name: "reasoning tokens exceed completion tokens",
			params: sqlc.CreateUsageRecordParams{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
				CachedTokens:     0,
				ReasoningTokens:  6,
				Source:           "upstream_response",
			},
		},
		{
			name: "invalid source",
			params: sqlc.CreateUsageRecordParams{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
				CachedTokens:     0,
				ReasoningTokens:  0,
				Source:           "estimated",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requestID := fmt.Sprintf("usage-invalid-%s-%d", tc.name, time.Now().UnixNano())
			requestRecord := createRequestRecordForTest(t, ctx, queries, identity, requestID)
			tc.params.RequestRecordID = requestRecord.ID

			_, err := queries.CreateUsageRecord(ctx, tc.params)
			if err == nil {
				t.Fatal("expected check violation")
			}
			if !isCheckViolation(err) {
				t.Fatalf("expected check violation, got %v", err)
			}
		})
	}
}
