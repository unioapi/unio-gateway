package ledger

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ThankCat/unio-api/internal/store/sqlc"
)

// numeric 创建测试用 NUMERIC 参数，避免 ledger 测试使用 float64。
func numeric(value int64) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(value), Valid: true}
}

// assertNumericEquals 校验 NUMERIC 字段的整数测试值。
func assertNumericEquals(t *testing.T, got pgtype.Numeric, want int64) {
	t.Helper()

	if !got.Valid {
		t.Fatalf("expected numeric %d to be valid", want)
	}
	if got.Exp != 0 {
		t.Fatalf("expected numeric exponent 0, got %d", got.Exp)
	}
	if got.Int.Cmp(big.NewInt(want)) != 0 {
		t.Fatalf("expected numeric %d, got %v", want, got.Int)
	}
}

// newServiceTestDeps 创建 ledger service 集成测试依赖。
func newServiceTestDeps(t *testing.T) (context.Context, *pgxpool.Pool, *sqlc.Queries, *Service, func()) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

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

	queries := sqlc.New(pool)
	service := NewService(pool, queries)
	cleanup := func() {
		pool.Close()
		cancel()
	}

	return ctx, pool, queries, service, cleanup
}

// createLedgerTestUser 创建 ledger service 测试用户。
func createLedgerTestUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int64 {
	t.Helper()

	email := fmt.Sprintf("ledger-service-%d@example.test", time.Now().UnixNano())

	var userID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING id
	`, email, "test-password-hash", "Ledger Test User").Scan(&userID)
	if err != nil {
		t.Fatalf("insert ledger test user: %v", err)
	}

	return userID
}

// cleanupLedgerTestUser 删除 ledger service 测试产生的账本、余额和用户。
func cleanupLedgerTestUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID int64) {
	t.Helper()

	if _, err := pool.Exec(ctx, `DELETE FROM ledger_entries WHERE user_id = $1`, userID); err != nil {
		t.Fatalf("delete ledger entries: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM user_balances WHERE user_id = $1`, userID); err != nil {
		t.Fatalf("delete user balances: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		t.Fatalf("delete user: %v", err)
	}
}

func TestCreditCreatesBalanceAndIsIdempotent(t *testing.T) {
	ctx, pool, queries, service, cleanup := newServiceTestDeps(t)
	defer cleanup()

	userID := createLedgerTestUser(t, ctx, pool)
	defer cleanupLedgerTestUser(t, context.Background(), pool, userID)

	params := CreditParams{
		UserID:         userID,
		Amount:         numeric(100),
		Currency:       "USD",
		IdempotencyKey: fmt.Sprintf("credit-%d", time.Now().UnixNano()),
		Reason:         "test credit",
	}

	created, err := service.Credit(ctx, params)
	if err != nil {
		t.Fatalf("credit: %v", err)
	}
	if created.EntryType != "credit" {
		t.Fatalf("expected credit entry type, got %q", created.EntryType)
	}
	if created.UserID != userID {
		t.Fatalf("expected user id %d, got %d", userID, created.UserID)
	}
	if created.RequestRecordID != nil {
		t.Fatalf("expected nil request record id, got %d", *created.RequestRecordID)
	}
	assertNumericEquals(t, created.Amount, 100)
	assertNumericEquals(t, created.BalanceBefore, 0)
	assertNumericEquals(t, created.BalanceAfter, 100)

	balance, err := queries.GetUserBalance(ctx, sqlc.GetUserBalanceParams{
		UserID:   userID,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("get balance after credit: %v", err)
	}
	assertNumericEquals(t, balance.Balance, 100)

	repeated, err := service.Credit(ctx, params)
	if err != nil {
		t.Fatalf("repeat credit: %v", err)
	}
	if repeated.ID != created.ID {
		t.Fatalf("expected idempotent credit entry id %d, got %d", created.ID, repeated.ID)
	}

	balance, err = queries.GetUserBalance(ctx, sqlc.GetUserBalanceParams{
		UserID:   userID,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("get balance after repeated credit: %v", err)
	}
	assertNumericEquals(t, balance.Balance, 100)

	entries, err := queries.ListLedgerEntriesByUser(ctx, sqlc.ListLedgerEntriesByUserParams{
		UserID:     userID,
		Currency:   "USD",
		LimitRows:  10,
		OffsetRows: 0,
	})
	if err != nil {
		t.Fatalf("list ledger entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 ledger entry after idempotent credit, got %d", len(entries))
	}
}

func TestDebitSubtractsBalanceAndIsIdempotent(t *testing.T) {
	ctx, pool, queries, service, cleanup := newServiceTestDeps(t)
	defer cleanup()

	userID := createLedgerTestUser(t, ctx, pool)
	defer cleanupLedgerTestUser(t, context.Background(), pool, userID)

	if _, err := service.Credit(ctx, CreditParams{
		UserID:         userID,
		Amount:         numeric(100),
		Currency:       "USD",
		IdempotencyKey: fmt.Sprintf("seed-credit-%d", time.Now().UnixNano()),
		Reason:         "seed balance",
	}); err != nil {
		t.Fatalf("seed credit: %v", err)
	}

	params := DebitParams{
		UserID:         userID,
		Amount:         numeric(35),
		Currency:       "USD",
		IdempotencyKey: fmt.Sprintf("debit-%d", time.Now().UnixNano()),
		Reason:         "test debit",
	}

	created, err := service.Debit(ctx, params)
	if err != nil {
		t.Fatalf("debit: %v", err)
	}
	if created.EntryType != "debit" {
		t.Fatalf("expected debit entry type, got %q", created.EntryType)
	}
	assertNumericEquals(t, created.Amount, 35)
	assertNumericEquals(t, created.BalanceBefore, 100)
	assertNumericEquals(t, created.BalanceAfter, 65)

	balance, err := queries.GetUserBalance(ctx, sqlc.GetUserBalanceParams{
		UserID:   userID,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("get balance after debit: %v", err)
	}
	assertNumericEquals(t, balance.Balance, 65)

	repeated, err := service.Debit(ctx, params)
	if err != nil {
		t.Fatalf("repeat debit: %v", err)
	}
	if repeated.ID != created.ID {
		t.Fatalf("expected idempotent debit entry id %d, got %d", created.ID, repeated.ID)
	}

	balance, err = queries.GetUserBalance(ctx, sqlc.GetUserBalanceParams{
		UserID:   userID,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("get balance after repeated debit: %v", err)
	}
	assertNumericEquals(t, balance.Balance, 65)
}

func TestIdempotencyKeyConflictReturnsDomainError(t *testing.T) {
	ctx, pool, _, service, cleanup := newServiceTestDeps(t)
	defer cleanup()

	userID := createLedgerTestUser(t, ctx, pool)
	defer cleanupLedgerTestUser(t, context.Background(), pool, userID)

	key := fmt.Sprintf("conflict-%d", time.Now().UnixNano())
	if _, err := service.Credit(ctx, CreditParams{
		UserID:         userID,
		Amount:         numeric(100),
		Currency:       "USD",
		IdempotencyKey: key,
		Reason:         "initial credit",
	}); err != nil {
		t.Fatalf("initial credit: %v", err)
	}

	_, err := service.Credit(ctx, CreditParams{
		UserID:         userID,
		Amount:         numeric(200),
		Currency:       "USD",
		IdempotencyKey: key,
		Reason:         "conflicting amount",
	})
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("expected ErrIdempotencyConflict for different amount, got %v", err)
	}

	_, err = service.Debit(ctx, DebitParams{
		UserID:         userID,
		Amount:         numeric(100),
		Currency:       "USD",
		IdempotencyKey: key,
		Reason:         "conflicting entry type",
	})
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("expected ErrIdempotencyConflict for different entry type, got %v", err)
	}
}

func TestConcurrentSameDebitIdempotencyKeyDoesNotDoubleCharge(t *testing.T) {
	ctx, pool, queries, service, cleanup := newServiceTestDeps(t)
	defer cleanup()

	userID := createLedgerTestUser(t, ctx, pool)
	defer cleanupLedgerTestUser(t, context.Background(), pool, userID)

	if _, err := service.Credit(ctx, CreditParams{
		UserID:         userID,
		Amount:         numeric(100),
		Currency:       "USD",
		IdempotencyKey: fmt.Sprintf("concurrent-seed-credit-%d", time.Now().UnixNano()),
		Reason:         "seed concurrent balance",
	}); err != nil {
		t.Fatalf("seed credit: %v", err)
	}

	params := DebitParams{
		UserID:         userID,
		Amount:         numeric(40),
		Currency:       "USD",
		IdempotencyKey: fmt.Sprintf("concurrent-debit-%d", time.Now().UnixNano()),
		Reason:         "concurrent debit",
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan Entry, 2)
	errs := make(chan error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			<-start
			entry, err := service.Debit(context.Background(), params)
			if err != nil {
				errs <- err
				return
			}
			results <- entry
		}()
	}

	close(start)
	wg.Wait()
	close(results)
	close(errs)

	if len(errs) != 0 {
		for err := range errs {
			t.Fatalf("concurrent debit failed: %v", err)
		}
	}

	var entries []Entry
	for entry := range results {
		entries = append(entries, entry)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 successful debit results, got %d", len(entries))
	}
	if entries[0].ID != entries[1].ID {
		t.Fatalf("expected concurrent idempotent debits to return same entry id, got %d and %d", entries[0].ID, entries[1].ID)
	}

	balance, err := queries.GetUserBalance(ctx, sqlc.GetUserBalanceParams{
		UserID:   userID,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("get balance after concurrent debit: %v", err)
	}
	assertNumericEquals(t, balance.Balance, 60)

	entriesByUser, err := queries.ListLedgerEntriesByUser(ctx, sqlc.ListLedgerEntriesByUserParams{
		UserID:     userID,
		Currency:   "USD",
		LimitRows:  10,
		OffsetRows: 0,
	})
	if err != nil {
		t.Fatalf("list ledger entries: %v", err)
	}
	if len(entriesByUser) != 2 {
		t.Fatalf("expected seed credit and one debit entry, got %d entries", len(entriesByUser))
	}
}

func TestDebitReturnsInsufficientBalanceWithoutLedgerEntry(t *testing.T) {
	ctx, pool, queries, service, cleanup := newServiceTestDeps(t)
	defer cleanup()

	userID := createLedgerTestUser(t, ctx, pool)
	defer cleanupLedgerTestUser(t, context.Background(), pool, userID)

	_, err := service.Debit(ctx, DebitParams{
		UserID:         userID,
		Amount:         numeric(1),
		Currency:       "USD",
		IdempotencyKey: fmt.Sprintf("insufficient-debit-%d", time.Now().UnixNano()),
		Reason:         "test insufficient balance",
	})
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}

	entries, err := queries.ListLedgerEntriesByUser(ctx, sqlc.ListLedgerEntriesByUserParams{
		UserID:     userID,
		Currency:   "USD",
		LimitRows:  10,
		OffsetRows: 0,
	})
	if err != nil {
		t.Fatalf("list ledger entries: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no ledger entries for insufficient balance, got %d", len(entries))
	}
}
