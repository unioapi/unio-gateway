package gateway

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/apikey"
	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/billing"
	"github.com/ThankCat/unio-api/internal/ledger"
	"github.com/ThankCat/unio-api/internal/requestlog"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// fakeChatBillingCalculator 是 chat settlement 测试使用的 billing calculator 替身。
type fakeChatBillingCalculator struct {
	usages     []billing.Usage
	prices     []billing.PriceSnapshot
	settlement billing.Settlement
	err        error
}

// fakeChatLedgerDebiter 是 chat settlement 测试使用的 ledger debiter 替身。
type fakeChatLedgerDebiter struct {
	params  []ledger.DebitParams
	queries []*sqlc.Queries
	err     error
}

// Calculate 记录 billing 入参，并返回测试预设结算结果。
func (c *fakeChatBillingCalculator) Calculate(usage billing.Usage, price billing.PriceSnapshot) (billing.Settlement, error) {
	c.usages = append(c.usages, usage)
	c.prices = append(c.prices, price)
	if c.err != nil {
		return billing.Settlement{}, c.err
	}

	return c.settlement, nil
}

// DebitWithQueries 记录事务内 ledger debit 参数，并返回测试预设错误。
func (d *fakeChatLedgerDebiter) DebitWithQueries(ctx context.Context, queries *sqlc.Queries, params ledger.DebitParams) (ledger.Entry, error) {
	d.queries = append(d.queries, queries)
	d.params = append(d.params, params)
	if d.err != nil {
		return ledger.Entry{}, d.err
	}

	return ledger.Entry{ID: 1, UserID: params.UserID, Amount: params.Amount, Currency: params.Currency}, nil
}

// chatSettlementDBDeps 保存 chat settlement 集成测试依赖。
type chatSettlementDBDeps struct {
	ctx           context.Context
	cancel        context.CancelFunc
	pool          *pgxpool.Pool
	queries       *sqlc.Queries
	userID        int64
	projectID     int64
	apiKeyID      int64
	providerID    int64
	channelID     int64
	modelID       int64
	priceID       int64
	requestRecord sqlc.RequestRecord
	attemptRecord sqlc.RequestAttempt
}

// newChatSettlementDBDeps 创建带真实数据库记录的 chat settlement 测试依赖。
func newChatSettlementDBDeps(t *testing.T) *chatSettlementDBDeps {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		cancel()
		t.Fatalf("create postgres pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		cancel()
		t.Fatalf("ping postgres: %v", err)
	}

	deps := &chatSettlementDBDeps{
		ctx:     ctx,
		cancel:  cancel,
		pool:    pool,
		queries: sqlc.New(pool),
	}
	t.Cleanup(func() {
		deps.cleanup()
	})

	deps.seed(t)

	return deps
}

// cleanup 删除测试提交的数据，避免污染本地开发库。
func (d *chatSettlementDBDeps) cleanup() {
	ctx := context.Background()

	if d.requestRecord.ID != 0 {
		_, _ = d.pool.Exec(ctx, `DELETE FROM ledger_entries WHERE request_record_id = $1`, d.requestRecord.ID)
		_, _ = d.pool.Exec(ctx, `DELETE FROM price_snapshots WHERE request_record_id = $1`, d.requestRecord.ID)
		_, _ = d.pool.Exec(ctx, `DELETE FROM usage_records WHERE request_record_id = $1`, d.requestRecord.ID)
		_, _ = d.pool.Exec(ctx, `DELETE FROM request_attempts WHERE request_record_id = $1`, d.requestRecord.ID)
		_, _ = d.pool.Exec(ctx, `DELETE FROM request_records WHERE id = $1`, d.requestRecord.ID)
	}
	if d.priceID != 0 {
		_, _ = d.pool.Exec(ctx, `DELETE FROM prices WHERE id = $1`, d.priceID)
	}
	if d.channelID != 0 {
		_, _ = d.pool.Exec(ctx, `DELETE FROM channels WHERE id = $1`, d.channelID)
	}
	if d.providerID != 0 {
		_, _ = d.pool.Exec(ctx, `DELETE FROM providers WHERE id = $1`, d.providerID)
	}
	if d.modelID != 0 {
		_, _ = d.pool.Exec(ctx, `DELETE FROM models WHERE id = $1`, d.modelID)
	}
	if d.apiKeyID != 0 {
		_, _ = d.pool.Exec(ctx, `DELETE FROM api_keys WHERE id = $1`, d.apiKeyID)
	}
	if d.projectID != 0 {
		_, _ = d.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, d.projectID)
	}
	if d.userID != 0 {
		_, _ = d.pool.Exec(ctx, `DELETE FROM ledger_entries WHERE user_id = $1`, d.userID)
		_, _ = d.pool.Exec(ctx, `DELETE FROM user_balances WHERE user_id = $1`, d.userID)
		_, _ = d.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, d.userID)
	}

	d.pool.Close()
	d.cancel()
}

// seed 插入一次可结算 chat 请求所需的身份、模型、价格、请求和余额数据。
func (d *chatSettlementDBDeps) seed(t *testing.T) {
	t.Helper()

	suffix := time.Now().UnixNano()

	user, err := d.queries.CreateUser(d.ctx, sqlc.CreateUserParams{
		Email:        fmt.Sprintf("chat-settlement-%d@example.com", suffix),
		PasswordHash: "test-password-hash",
		DisplayName:  "Chat Settlement User",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	d.userID = user.ID

	project, err := d.queries.CreateProject(d.ctx, sqlc.CreateProjectParams{
		UserID: user.ID,
		Name:   fmt.Sprintf("chat-settlement-project-%d", suffix),
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	d.projectID = project.ID

	generatedKey, err := apikey.Generate()
	if err != nil {
		t.Fatalf("generate api key: %v", err)
	}
	apiKey, err := d.queries.CreateAPIKey(d.ctx, sqlc.CreateAPIKeyParams{
		ProjectID: project.ID,
		Name:      "chat settlement key",
		KeyPrefix: generatedKey.Prefix,
		KeyHash:   generatedKey.Hash,
		ExpiresAt: pgtype.Timestamptz{Valid: false},
	})
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}
	d.apiKeyID = apiKey.ID

	d.providerID = insertChatSettlementProvider(t, d.ctx, d.pool, suffix)
	d.channelID = insertChatSettlementChannel(t, d.ctx, d.pool, d.providerID, suffix)
	d.modelID = insertChatSettlementModel(t, d.ctx, d.pool, suffix)

	price, err := d.queries.CreatePrice(d.ctx, sqlc.CreatePriceParams{
		ModelID:              d.modelID,
		Currency:             "USD",
		PricingUnit:          billing.PricingUnitPer1MTokens,
		InputPrice:           testNumeric(2_0000000000, -10),
		OutputPrice:          testNumeric(8_0000000000, -10),
		CachedInputPrice:     testNumeric(5000000000, -10),
		ReasoningOutputPrice: testNumeric(12_0000000000, -10),
		Status:               "enabled",
		EffectiveFrom:        pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
		EffectiveTo:          pgtype.Timestamptz{Valid: false},
	})
	if err != nil {
		t.Fatalf("create price: %v", err)
	}
	d.priceID = price.ID

	requestRecord, err := d.queries.CreateRequestRecord(d.ctx, sqlc.CreateRequestRecordParams{
		RequestID:        fmt.Sprintf("chat-settlement-request-%d", suffix),
		UserID:           user.ID,
		ProjectID:        project.ID,
		ApiKeyID:         apiKey.ID,
		RequestedModelID: "openai/gpt-4.1",
		ResponseModelID:  pgtype.Text{Valid: false},
		Stream:           false,
		Status:           string(requestlog.RequestStatusRunning),
		FinalProviderID:  pgtype.Int8{Valid: false},
		FinalChannelID:   pgtype.Int8{Valid: false},
		ErrorCode:        pgtype.Text{Valid: false},
		ErrorMessage:     pgtype.Text{Valid: false},
		StartedAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
		CompletedAt:      pgtype.Timestamptz{Valid: false},
	})
	if err != nil {
		t.Fatalf("create request record: %v", err)
	}
	d.requestRecord = requestRecord

	attemptRecord, err := d.queries.CreateRequestAttempt(d.ctx, sqlc.CreateRequestAttemptParams{
		RequestRecordID:       requestRecord.ID,
		AttemptIndex:          0,
		ProviderID:            d.providerID,
		ChannelID:             d.channelID,
		AdapterKey:            "openai",
		UpstreamModel:         "gpt-4.1",
		UpstreamResponseModel: pgtype.Text{Valid: false},
		Status:                string(requestlog.AttemptStatusRunning),
		UpstreamStatusCode:    pgtype.Int4{Valid: false},
		UpstreamRequestID:     pgtype.Text{Valid: false},
		ErrorCode:             pgtype.Text{Valid: false},
		ErrorMessage:          pgtype.Text{Valid: false},
		StartedAt:             pgtype.Timestamptz{Time: time.Now(), Valid: true},
		CompletedAt:           pgtype.Timestamptz{Valid: false},
	})
	if err != nil {
		t.Fatalf("create request attempt: %v", err)
	}
	d.attemptRecord = attemptRecord

	if err := d.queries.EnsureUserBalance(d.ctx, sqlc.EnsureUserBalanceParams{
		UserID:   user.ID,
		Currency: "USD",
	}); err != nil {
		t.Fatalf("ensure user balance: %v", err)
	}
	if _, err := d.queries.AddUserBalance(d.ctx, sqlc.AddUserBalanceParams{
		Amount:   testNumeric(10_0000000000, -10),
		UserID:   user.ID,
		Currency: "USD",
	}); err != nil {
		t.Fatalf("add user balance: %v", err)
	}
}

// params 创建 chat settlement 测试参数。
func (d *chatSettlementDBDeps) params() ChatSettlementParams {
	return ChatSettlementParams{
		RequestRecord: requestlog.RequestRecord{
			ID:               d.requestRecord.ID,
			UserID:           d.userID,
			ProjectID:        d.projectID,
			APIKeyID:         d.apiKeyID,
			RequestedModelID: d.requestRecord.RequestedModelID,
			Status:           requestlog.RequestStatus(d.requestRecord.Status),
		},
		AttemptRecord: requestlog.AttemptRecord{
			ID:              d.attemptRecord.ID,
			RequestRecordID: d.requestRecord.ID,
			AttemptIndex:    int(d.attemptRecord.AttemptIndex),
			ProviderID:      d.providerID,
			ChannelID:       d.channelID,
			AdapterKey:      d.attemptRecord.AdapterKey,
			UpstreamModel:   d.attemptRecord.UpstreamModel,
			Status:          requestlog.AttemptStatus(d.attemptRecord.Status),
		},
		Principal:       &auth.APIKeyPrincipal{UserID: d.userID, ProjectID: d.projectID, APIKeyID: d.apiKeyID},
		ResponseModelID: "openai/gpt-4.1",
		ModelDBID:       d.modelID,
		FinalProviderID: d.providerID,
		FinalChannelID:  d.channelID,
		AdapterResp: &adapter.ChatResponse{
			ID:      "chatcmpl_test",
			Model:   "gpt-4.1",
			Content: "hello",
			Usage: adapter.ChatUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		},
	}
}

// insertChatSettlementProvider 插入测试 provider。
func insertChatSettlementProvider(t *testing.T, ctx context.Context, pool *pgxpool.Pool, suffix int64) int64 {
	t.Helper()

	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO providers (slug, name, adapter, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, fmt.Sprintf("chat-settlement-provider-%d", suffix), "Chat Settlement Provider", "openai", "enabled").Scan(&id)
	if err != nil {
		t.Fatalf("insert provider: %v", err)
	}

	return id
}

// insertChatSettlementChannel 插入测试 channel。
func insertChatSettlementChannel(t *testing.T, ctx context.Context, pool *pgxpool.Pool, providerID int64, suffix int64) int64 {
	t.Helper()

	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO channels (provider_id, name, base_url, credential_ref, status, priority, timeout_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, providerID, fmt.Sprintf("chat-settlement-channel-%d", suffix), "https://example.test/v1", "secret://chat-settlement", "enabled", 10, 30000).Scan(&id)
	if err != nil {
		t.Fatalf("insert channel: %v", err)
	}

	return id
}

// insertChatSettlementModel 插入测试 model。
func insertChatSettlementModel(t *testing.T, ctx context.Context, pool *pgxpool.Pool, suffix int64) int64 {
	t.Helper()

	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO models (model_id, display_name, owned_by, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, fmt.Sprintf("chat-settlement-model-%d", suffix), "Chat Settlement Model", "openai", "enabled").Scan(&id)
	if err != nil {
		t.Fatalf("insert model: %v", err)
	}

	return id
}

// chatSettlementBilling 创建测试用 billing calculator。
func chatSettlementBilling(amount pgtype.Numeric) *fakeChatBillingCalculator {
	return &fakeChatBillingCalculator{
		settlement: billing.Settlement{
			Amount:         amount,
			Currency:       "USD",
			FormulaVersion: billing.FormulaVersionV1,
		},
	}
}

// requestStatus 查询 request record 当前状态。
func requestStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, requestRecordID int64) string {
	t.Helper()

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM request_records WHERE id = $1`, requestRecordID).Scan(&status); err != nil {
		t.Fatalf("query request status: %v", err)
	}

	return status
}

// attemptStatus 查询 request attempt 当前状态。
func attemptStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, attemptID int64) string {
	t.Helper()

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM request_attempts WHERE id = $1`, attemptID).Scan(&status); err != nil {
		t.Fatalf("query attempt status: %v", err)
	}

	return status
}

// testNumeric 创建测试用 pgtype.Numeric。
func testNumeric(value int64, exp int32) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(value), Exp: exp, Valid: true}
}

func TestChatSettlementSettlesSuccessfulChat(t *testing.T) {
	deps := newChatSettlementDBDeps(t)
	billingCalculator := chatSettlementBilling(testNumeric(61_000000, -10))
	ledgerService := ledger.NewService(deps.pool, deps.queries)
	service := NewChatSettlementService(deps.pool, deps.queries, billingCalculator, ledgerService)

	if err := service.SettleSuccessfulChat(deps.ctx, deps.params()); err != nil {
		t.Fatalf("settle successful chat: %v", err)
	}

	usageRecord, err := deps.queries.GetUsageRecordByRequest(deps.ctx, deps.requestRecord.ID)
	if err != nil {
		t.Fatalf("get usage record: %v", err)
	}
	if usageRecord.PromptTokens != 10 || usageRecord.CompletionTokens != 5 || usageRecord.TotalTokens != 15 {
		t.Fatalf("expected usage 10/5/15, got %d/%d/%d", usageRecord.PromptTokens, usageRecord.CompletionTokens, usageRecord.TotalTokens)
	}
	if usageRecord.Source != "upstream_response" {
		t.Fatalf("expected source upstream_response, got %q", usageRecord.Source)
	}

	snapshot, err := deps.queries.GetPriceSnapshotByRequest(deps.ctx, deps.requestRecord.ID)
	if err != nil {
		t.Fatalf("get price snapshot: %v", err)
	}
	if !snapshot.PriceID.Valid || snapshot.PriceID.Int64 != deps.priceID {
		t.Fatalf("expected snapshot price id %d, got valid=%v value=%d", deps.priceID, snapshot.PriceID.Valid, snapshot.PriceID.Int64)
	}
	if snapshot.FormulaVersion != billing.FormulaVersionV1 {
		t.Fatalf("expected formula version %q, got %q", billing.FormulaVersionV1, snapshot.FormulaVersion)
	}

	entry, err := deps.queries.GetLedgerEntryByIdempotencyKey(deps.ctx, fmt.Sprintf("chat:settle:%d", deps.requestRecord.ID))
	if err != nil {
		t.Fatalf("get ledger entry: %v", err)
	}
	if entry.UserID != deps.userID || entry.EntryType != "debit" {
		t.Fatalf("unexpected ledger entry user/type %d/%q", entry.UserID, entry.EntryType)
	}

	if status := requestStatus(t, deps.ctx, deps.pool, deps.requestRecord.ID); status != string(requestlog.RequestStatusSucceeded) {
		t.Fatalf("expected request succeeded, got %q", status)
	}
	if status := attemptStatus(t, deps.ctx, deps.pool, deps.attemptRecord.ID); status != string(requestlog.AttemptStatusSucceeded) {
		t.Fatalf("expected attempt succeeded, got %q", status)
	}

	if len(billingCalculator.usages) != 1 || len(billingCalculator.prices) != 1 {
		t.Fatalf("expected one billing calculation, got usages=%d prices=%d", len(billingCalculator.usages), len(billingCalculator.prices))
	}
}

func TestChatSettlementSkipsLedgerDebitForZeroAmount(t *testing.T) {
	deps := newChatSettlementDBDeps(t)
	billingCalculator := chatSettlementBilling(testNumeric(0, -10))
	ledgerDebiter := &fakeChatLedgerDebiter{}
	service := NewChatSettlementService(deps.pool, deps.queries, billingCalculator, ledgerDebiter)

	if err := service.SettleSuccessfulChat(deps.ctx, deps.params()); err != nil {
		t.Fatalf("settle successful chat: %v", err)
	}

	if len(ledgerDebiter.params) != 0 {
		t.Fatalf("expected no ledger debit for zero amount, got %d", len(ledgerDebiter.params))
	}
	if status := requestStatus(t, deps.ctx, deps.pool, deps.requestRecord.ID); status != string(requestlog.RequestStatusSucceeded) {
		t.Fatalf("expected request succeeded, got %q", status)
	}
	if _, err := deps.queries.GetUsageRecordByRequest(deps.ctx, deps.requestRecord.ID); err != nil {
		t.Fatalf("expected committed usage record: %v", err)
	}
	if _, err := deps.queries.GetPriceSnapshotByRequest(deps.ctx, deps.requestRecord.ID); err != nil {
		t.Fatalf("expected committed price snapshot: %v", err)
	}
}

func TestChatSettlementRollsBackFactsWhenLedgerDebitFails(t *testing.T) {
	deps := newChatSettlementDBDeps(t)
	ledgerErr := errors.New("ledger debit failed")
	ledgerDebiter := &fakeChatLedgerDebiter{err: ledgerErr}
	service := NewChatSettlementService(
		deps.pool,
		deps.queries,
		chatSettlementBilling(testNumeric(61_000000, -10)),
		ledgerDebiter,
	)

	err := service.SettleSuccessfulChat(deps.ctx, deps.params())
	if !errors.Is(err, ledgerErr) {
		t.Fatalf("expected ledger error, got %v", err)
	}
	if len(ledgerDebiter.params) != 1 {
		t.Fatalf("expected one ledger debit attempt, got %d", len(ledgerDebiter.params))
	}

	if _, err := deps.queries.GetUsageRecordByRequest(deps.ctx, deps.requestRecord.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected usage record rollback, got %v", err)
	}
	if _, err := deps.queries.GetPriceSnapshotByRequest(deps.ctx, deps.requestRecord.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected price snapshot rollback, got %v", err)
	}
	if status := requestStatus(t, deps.ctx, deps.pool, deps.requestRecord.ID); status != string(requestlog.RequestStatusRunning) {
		t.Fatalf("expected request to remain running after rollback, got %q", status)
	}
	if status := attemptStatus(t, deps.ctx, deps.pool, deps.attemptRecord.ID); status != string(requestlog.AttemptStatusRunning) {
		t.Fatalf("expected attempt to remain running after rollback, got %q", status)
	}
}

func TestChatSettlementRollsBackFactsWhenBillingFails(t *testing.T) {
	deps := newChatSettlementDBDeps(t)
	billingErr := errors.New("billing calculation failed")
	billingCalculator := chatSettlementBilling(testNumeric(61_000000, -10))
	billingCalculator.err = billingErr
	ledgerDebiter := &fakeChatLedgerDebiter{}
	service := NewChatSettlementService(deps.pool, deps.queries, billingCalculator, ledgerDebiter)

	err := service.SettleSuccessfulChat(deps.ctx, deps.params())
	if !errors.Is(err, billingErr) {
		t.Fatalf("expected billing error, got %v", err)
	}
	if len(ledgerDebiter.params) != 0 {
		t.Fatalf("expected no ledger debit after billing error, got %d", len(ledgerDebiter.params))
	}
	if _, err := deps.queries.GetUsageRecordByRequest(deps.ctx, deps.requestRecord.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected usage record rollback, got %v", err)
	}
	if _, err := deps.queries.GetPriceSnapshotByRequest(deps.ctx, deps.requestRecord.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected price snapshot rollback, got %v", err)
	}
}
