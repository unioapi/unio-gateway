package sqlc_test

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ThankCat/unio-api/internal/store/sqlc"
)

// isForeignKeyViolation 判断数据库错误是否是外键约束冲突。
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// assertNumericEquals 校验 NUMERIC 字段表示的金额值，忽略 PostgreSQL 返回的 scale 差异。
func assertNumericEquals(t *testing.T, got pgtype.Numeric, want int64) {
	t.Helper()

	if !got.Valid {
		t.Fatalf("expected numeric %d to be valid", want)
	}
	if got.Int == nil {
		t.Fatal("expected numeric int to be set")
	}

	rat := new(big.Rat).SetInt(new(big.Int).Set(got.Int))
	if got.Exp > 0 {
		rat.Mul(rat, new(big.Rat).SetInt(pow10(got.Exp)))
	}
	if got.Exp < 0 {
		rat.Quo(rat, new(big.Rat).SetInt(pow10(-got.Exp)))
	}

	if rat.Cmp(big.NewRat(want, 1)) != 0 {
		t.Fatalf("expected numeric %d, got %s", want, rat.String())
	}
}

// pow10 返回 10 的 exp 次方。
func pow10(exp int32) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil)
}

// createLedgerEntryForTest 创建测试账本流水。
func createLedgerEntryForTest(t *testing.T, queries *sqlc.Queries, ctx context.Context, params sqlc.CreateLedgerEntryParams) sqlc.LedgerEntry {
	t.Helper()

	entry, err := queries.CreateLedgerEntry(ctx, params)
	if err != nil {
		t.Fatalf("create ledger entry: %v", err)
	}

	return entry
}

func TestUserBalanceLifecycleAndUniqueness(t *testing.T) {
	ctx, _, queries, cleanup := newModelChannelTestTx(t)
	defer cleanup()

	identity := createRequestRecordIdentity(t, ctx, queries)

	created, err := queries.CreateUserBalance(ctx, sqlc.CreateUserBalanceParams{
		UserID:   identity.user.ID,
		Currency: "USD",
		Balance:  numeric(100),
	})
	if err != nil {
		t.Fatalf("create user balance: %v", err)
	}
	if created.UserID != identity.user.ID {
		t.Fatalf("expected user id %d, got %d", identity.user.ID, created.UserID)
	}
	if created.Currency != "USD" {
		t.Fatalf("expected currency USD, got %q", created.Currency)
	}
	assertNumericEquals(t, created.Balance, 100)
	if !created.CreatedAt.Valid || !created.UpdatedAt.Valid {
		t.Fatal("expected balance timestamps to be set")
	}

	got, err := queries.GetUserBalance(ctx, sqlc.GetUserBalanceParams{
		UserID:   identity.user.ID,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("get user balance: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected balance id %d, got %d", created.ID, got.ID)
	}

	locked, err := queries.GetUserBalanceForUpdate(ctx, sqlc.GetUserBalanceForUpdateParams{
		UserID:   identity.user.ID,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("get user balance for update: %v", err)
	}
	if locked.ID != created.ID {
		t.Fatalf("expected locked balance id %d, got %d", created.ID, locked.ID)
	}

	updated, err := queries.UpdateUserBalance(ctx, sqlc.UpdateUserBalanceParams{
		Balance:  numeric(75),
		UserID:   identity.user.ID,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("update user balance: %v", err)
	}
	assertNumericEquals(t, updated.Balance, 75)
	if !updated.UpdatedAt.Valid {
		t.Fatal("expected updated_at to be set")
	}

	_, err = queries.CreateUserBalance(ctx, sqlc.CreateUserBalanceParams{
		UserID:   identity.user.ID,
		Currency: "USD",
		Balance:  numeric(1),
	})
	if err == nil {
		t.Fatal("expected duplicate user currency balance to fail")
	}
	if !isUniqueViolation(err) {
		t.Fatalf("expected unique violation, got %v", err)
	}
}

func TestEnsureAddAndSubtractUserBalance(t *testing.T) {
	ctx, _, queries, cleanup := newModelChannelTestTx(t)
	defer cleanup()

	identity := createRequestRecordIdentity(t, ctx, queries)

	if err := queries.EnsureUserBalance(ctx, sqlc.EnsureUserBalanceParams{
		UserID:   identity.user.ID,
		Currency: "USD",
	}); err != nil {
		t.Fatalf("ensure user balance: %v", err)
	}

	// EnsureUserBalance 是幂等操作；余额行已存在时不应报唯一约束错误。
	if err := queries.EnsureUserBalance(ctx, sqlc.EnsureUserBalanceParams{
		UserID:   identity.user.ID,
		Currency: "USD",
	}); err != nil {
		t.Fatalf("ensure existing user balance: %v", err)
	}

	initial, err := queries.GetUserBalance(ctx, sqlc.GetUserBalanceParams{
		UserID:   identity.user.ID,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("get ensured user balance: %v", err)
	}
	assertNumericEquals(t, initial.Balance, 0)

	added, err := queries.AddUserBalance(ctx, sqlc.AddUserBalanceParams{
		Amount:   numeric(25),
		UserID:   identity.user.ID,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("add user balance: %v", err)
	}
	assertNumericEquals(t, added.Balance, 25)

	subtracted, err := queries.SubtractUserBalance(ctx, sqlc.SubtractUserBalanceParams{
		Amount:   numeric(10),
		UserID:   identity.user.ID,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("subtract user balance: %v", err)
	}
	assertNumericEquals(t, subtracted.Balance, 15)

	_, err = queries.SubtractUserBalance(ctx, sqlc.SubtractUserBalanceParams{
		Amount:   numeric(99),
		UserID:   identity.user.ID,
		Currency: "USD",
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected no rows for insufficient balance, got %v", err)
	}

	got, err := queries.GetUserBalance(ctx, sqlc.GetUserBalanceParams{
		UserID:   identity.user.ID,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("get user balance after failed subtract: %v", err)
	}
	assertNumericEquals(t, got.Balance, 15)
}

func TestUserBalanceRejectsInvalidConstraints(t *testing.T) {
	cases := []struct {
		name     string
		currency string
		balance  pgtype.Numeric
	}{
		{
			name:     "empty currency",
			currency: "",
			balance:  numeric(0),
		},
		{
			name:     "negative balance",
			currency: "USD",
			balance:  numeric(-1),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _, queries, cleanup := newModelChannelTestTx(t)
			defer cleanup()

			identity := createRequestRecordIdentity(t, ctx, queries)

			_, err := queries.CreateUserBalance(ctx, sqlc.CreateUserBalanceParams{
				UserID:   identity.user.ID,
				Currency: tc.currency,
				Balance:  tc.balance,
			})
			if err == nil {
				t.Fatal("expected check violation")
			}
			if !isCheckViolation(err) {
				t.Fatalf("expected check violation, got %v", err)
			}
		})
	}
}

func TestLedgerEntryCreateGetAndList(t *testing.T) {
	ctx, _, queries, cleanup := newModelChannelTestTx(t)
	defer cleanup()

	identity := createRequestRecordIdentity(t, ctx, queries)
	requestRecord := createRequestRecordForTest(t, ctx, queries, identity, fmt.Sprintf("ledger-list-%d", time.Now().UnixNano()))

	credit := createLedgerEntryForTest(t, queries, ctx, sqlc.CreateLedgerEntryParams{
		UserID:          identity.user.ID,
		RequestRecordID: pgtype.Int8{Valid: false},
		EntryType:       "credit",
		Amount:          numeric(100),
		Currency:        "USD",
		BalanceBefore:   numeric(0),
		BalanceAfter:    numeric(100),
		IdempotencyKey:  fmt.Sprintf("ledger-credit-%d", time.Now().UnixNano()),
		Reason:          "test credit",
	})
	debit := createLedgerEntryForTest(t, queries, ctx, sqlc.CreateLedgerEntryParams{
		UserID:          identity.user.ID,
		RequestRecordID: pgtype.Int8{Int64: requestRecord.ID, Valid: true},
		EntryType:       "debit",
		Amount:          numeric(25),
		Currency:        "USD",
		BalanceBefore:   numeric(100),
		BalanceAfter:    numeric(75),
		IdempotencyKey:  fmt.Sprintf("ledger-debit-%d", time.Now().UnixNano()),
		Reason:          "test debit",
	})
	refund := createLedgerEntryForTest(t, queries, ctx, sqlc.CreateLedgerEntryParams{
		UserID:          identity.user.ID,
		RequestRecordID: pgtype.Int8{Int64: requestRecord.ID, Valid: true},
		EntryType:       "refund",
		Amount:          numeric(5),
		Currency:        "USD",
		BalanceBefore:   numeric(75),
		BalanceAfter:    numeric(80),
		IdempotencyKey:  fmt.Sprintf("ledger-refund-%d", time.Now().UnixNano()),
		Reason:          "test refund",
	})

	got, err := queries.GetLedgerEntryByIdempotencyKey(ctx, debit.IdempotencyKey)
	if err != nil {
		t.Fatalf("get ledger entry by idempotency key: %v", err)
	}
	if got.ID != debit.ID {
		t.Fatalf("expected ledger id %d, got %d", debit.ID, got.ID)
	}
	assertNumericEquals(t, got.Amount, 25)
	assertNumericEquals(t, got.BalanceBefore, 100)
	assertNumericEquals(t, got.BalanceAfter, 75)

	byUser, err := queries.ListLedgerEntriesByUser(ctx, sqlc.ListLedgerEntriesByUserParams{
		UserID:     identity.user.ID,
		Currency:   "USD",
		OffsetRows: 0,
		LimitRows:  10,
	})
	if err != nil {
		t.Fatalf("list ledger entries by user: %v", err)
	}
	if len(byUser) != 3 {
		t.Fatalf("expected 3 ledger entries by user, got %d", len(byUser))
	}
	if byUser[0].ID != refund.ID || byUser[1].ID != debit.ID || byUser[2].ID != credit.ID {
		t.Fatalf("expected entries ordered refund/debit/credit, got ids %d/%d/%d", byUser[0].ID, byUser[1].ID, byUser[2].ID)
	}

	byRequest, err := queries.ListLedgerEntriesByRequest(ctx, pgtype.Int8{Int64: requestRecord.ID, Valid: true})
	if err != nil {
		t.Fatalf("list ledger entries by request: %v", err)
	}
	if len(byRequest) != 2 {
		t.Fatalf("expected 2 ledger entries by request, got %d", len(byRequest))
	}
	if byRequest[0].ID != debit.ID || byRequest[1].ID != refund.ID {
		t.Fatalf("expected request entries ordered debit/refund, got ids %d/%d", byRequest[0].ID, byRequest[1].ID)
	}
}

func TestLedgerEntryRejectsInvalidConstraints(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*sqlc.CreateLedgerEntryParams)
	}{
		{
			name: "invalid entry type",
			mutate: func(params *sqlc.CreateLedgerEntryParams) {
				params.EntryType = "reserve"
			},
		},
		{
			name: "zero amount",
			mutate: func(params *sqlc.CreateLedgerEntryParams) {
				params.Amount = numeric(0)
			},
		},
		{
			name: "negative amount",
			mutate: func(params *sqlc.CreateLedgerEntryParams) {
				params.Amount = numeric(-1)
			},
		},
		{
			name: "empty currency",
			mutate: func(params *sqlc.CreateLedgerEntryParams) {
				params.Currency = ""
			},
		},
		{
			name: "negative balance before",
			mutate: func(params *sqlc.CreateLedgerEntryParams) {
				params.BalanceBefore = numeric(-1)
			},
		},
		{
			name: "negative balance after",
			mutate: func(params *sqlc.CreateLedgerEntryParams) {
				params.BalanceAfter = numeric(-1)
			},
		},
		{
			name: "empty idempotency key",
			mutate: func(params *sqlc.CreateLedgerEntryParams) {
				params.IdempotencyKey = ""
			},
		},
		{
			name: "empty reason",
			mutate: func(params *sqlc.CreateLedgerEntryParams) {
				params.Reason = ""
			},
		},
		{
			name: "credit balance math mismatch",
			mutate: func(params *sqlc.CreateLedgerEntryParams) {
				params.EntryType = "credit"
				params.Amount = numeric(10)
				params.BalanceBefore = numeric(100)
				params.BalanceAfter = numeric(100)
			},
		},
		{
			name: "debit balance math mismatch",
			mutate: func(params *sqlc.CreateLedgerEntryParams) {
				params.EntryType = "debit"
				params.Amount = numeric(10)
				params.BalanceBefore = numeric(100)
				params.BalanceAfter = numeric(100)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _, queries, cleanup := newModelChannelTestTx(t)
			defer cleanup()

			identity := createRequestRecordIdentity(t, ctx, queries)
			params := sqlc.CreateLedgerEntryParams{
				UserID:          identity.user.ID,
				RequestRecordID: pgtype.Int8{Valid: false},
				EntryType:       "credit",
				Amount:          numeric(1),
				Currency:        "USD",
				BalanceBefore:   numeric(0),
				BalanceAfter:    numeric(1),
				IdempotencyKey:  fmt.Sprintf("ledger-invalid-%s-%d", tc.name, time.Now().UnixNano()),
				Reason:          "test invalid constraint",
			}
			tc.mutate(&params)

			_, err := queries.CreateLedgerEntry(ctx, params)
			if err == nil {
				t.Fatal("expected check violation")
			}
			if !isCheckViolation(err) {
				t.Fatalf("expected check violation, got %v", err)
			}
		})
	}
}

func TestLedgerEntryRejectsDuplicateIdempotencyKey(t *testing.T) {
	ctx, _, queries, cleanup := newModelChannelTestTx(t)
	defer cleanup()

	identity := createRequestRecordIdentity(t, ctx, queries)
	key := fmt.Sprintf("ledger-duplicate-%d", time.Now().UnixNano())
	params := sqlc.CreateLedgerEntryParams{
		UserID:          identity.user.ID,
		RequestRecordID: pgtype.Int8{Valid: false},
		EntryType:       "credit",
		Amount:          numeric(1),
		Currency:        "USD",
		BalanceBefore:   numeric(0),
		BalanceAfter:    numeric(1),
		IdempotencyKey:  key,
		Reason:          "test duplicate idempotency key",
	}

	createLedgerEntryForTest(t, queries, ctx, params)

	_, err := queries.CreateLedgerEntry(ctx, params)
	if err == nil {
		t.Fatal("expected duplicate idempotency key to fail")
	}
	if !isUniqueViolation(err) {
		t.Fatalf("expected unique violation, got %v", err)
	}
}

func TestLedgerEntryRequiresMatchingRequestUser(t *testing.T) {
	ctx, _, queries, cleanup := newModelChannelTestTx(t)
	defer cleanup()

	requestOwner := createRequestRecordIdentity(t, ctx, queries)
	otherUser := createRequestRecordIdentity(t, ctx, queries)
	requestRecord := createRequestRecordForTest(t, ctx, queries, requestOwner, fmt.Sprintf("ledger-request-owner-%d", time.Now().UnixNano()))

	_, err := queries.CreateLedgerEntry(ctx, sqlc.CreateLedgerEntryParams{
		UserID:          otherUser.user.ID,
		RequestRecordID: pgtype.Int8{Int64: requestRecord.ID, Valid: true},
		EntryType:       "debit",
		Amount:          numeric(1),
		Currency:        "USD",
		BalanceBefore:   numeric(10),
		BalanceAfter:    numeric(9),
		IdempotencyKey:  fmt.Sprintf("ledger-user-mismatch-%d", time.Now().UnixNano()),
		Reason:          "mismatched request user",
	})
	if err == nil {
		t.Fatal("expected ledger entry with mismatched request user to fail")
	}
	if !isForeignKeyViolation(err) {
		t.Fatalf("expected foreign key violation, got %v", err)
	}
}
