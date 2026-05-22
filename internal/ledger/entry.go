package ledger

import (
	"context"
	"errors"

	"github.com/ThankCat/unio-api/internal/failure"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Entry 表示 ledger service 返回给调用方的账本流水。
type Entry struct {
	ID              int64
	UserID          int64
	RequestRecordID *int64
	EntryType       EntryType
	Amount          pgtype.Numeric
	Currency        string
	BalanceBefore   pgtype.Numeric
	BalanceAfter    pgtype.Numeric
	IdempotencyKey  string
	Reason          string
}

// CreditParams 表示加款类账本操作参数。
type CreditParams struct {
	UserID          int64
	RequestRecordID *int64
	Amount          pgtype.Numeric
	Currency        string
	IdempotencyKey  string
	Reason          string
}

// DebitParams 表示扣款类账本操作参数。
type DebitParams struct {
	UserID          int64
	RequestRecordID *int64
	Amount          pgtype.Numeric
	Currency        string
	IdempotencyKey  string
	Reason          string
}

// Credit 增加用户余额，并在同一个事务里写入 credit 账本流水。
func (s *Service) Credit(ctx context.Context, params CreditParams) (Entry, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "begin ledger transaction")
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := s.queries.WithTx(tx)

	// 幂等命中表示这笔加款已经完成，直接返回已有流水，避免重复加余额。
	existing, err := txQueries.GetLedgerEntryByIdempotencyKey(ctx, params.IdempotencyKey)
	if err == nil {
		if err := ensureIdempotentEntryMatches(existing, params.UserID, params.RequestRecordID, EntryTypeCredit, params.Amount, params.Currency); err != nil {
			return Entry{}, err
		}

		return entryFromSQLC(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "lookup ledger idempotency key")
	}

	// Credit 可以为新用户创建 0 余额行，再在同一事务中加款。
	if err := txQueries.EnsureUserBalance(ctx, sqlc.EnsureUserBalanceParams{
		UserID:   params.UserID,
		Currency: params.Currency,
	}); err != nil {
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "ensure user balance")
	}

	// 锁定用户余额行，确保并发充值/扣费不会基于同一个旧余额计算。
	before, err := txQueries.GetUserBalanceForUpdate(ctx, sqlc.GetUserBalanceForUpdateParams{
		UserID:   params.UserID,
		Currency: params.Currency,
	})
	if err != nil {
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "lock user balance")
	}

	// 让 PostgreSQL 执行 NUMERIC 加法，避免在 Go 中用 float 或手写 decimal 计算金额。
	after, err := txQueries.AddUserBalance(ctx, sqlc.AddUserBalanceParams{
		Amount:   params.Amount,
		UserID:   params.UserID,
		Currency: params.Currency,
	})
	if err != nil {
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "add user balance")
	}

	// 写入账本事实；balance_before/after 必须和余额更新结果一致。
	created, err := txQueries.CreateLedgerEntry(ctx, sqlc.CreateLedgerEntryParams{
		UserID:          params.UserID,
		RequestRecordID: int64PtrToPgtypeInt8(params.RequestRecordID),
		EntryType:       string(EntryTypeCredit),
		Amount:          params.Amount,
		Currency:        params.Currency,
		BalanceBefore:   before.Balance,
		BalanceAfter:    after.Balance,
		IdempotencyKey:  params.IdempotencyKey,
		Reason:          params.Reason,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return s.resolveIdempotentCreateConflict(ctx, tx, params.IdempotencyKey, params.UserID, params.RequestRecordID, EntryTypeCredit, params.Amount, params.Currency)
		}

		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "create ledger entry")
	}

	if err := tx.Commit(ctx); err != nil {
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "commit ledger transaction")
	}

	return entryFromSQLC(created), nil
}

// Debit 减少用户余额，并在同一个事务里写入 debit 账本流水。
func (s *Service) Debit(ctx context.Context, params DebitParams) (Entry, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "begin ledger transaction")
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := s.queries.WithTx(tx)

	entry, err := s.debitWithQueries(ctx, txQueries, params)
	if err != nil {
		if isUniqueViolation(err) {
			return s.resolveIdempotentCreateConflict(ctx, tx, params.IdempotencyKey, params.UserID, params.RequestRecordID, EntryTypeDebit, params.Amount, params.Currency)
		}

		return Entry{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "commit ledger transaction")
	}

	return entry, nil
}

// DebitWithQueries 使用调用方传入的 queries 执行扣款。
// queries 可以是普通 queries，也可以是 queries.WithTx(tx)。
func (s *Service) DebitWithQueries(ctx context.Context, queries *sqlc.Queries, params DebitParams) (Entry, error) {
	// TODO(阶段7/production): [GAP-7-012] 外部事务内并发使用同一 debit 幂等键时，CreateLedgerEntry 唯一冲突会使调用方事务失败且无法在当前事务内安全查询既有流水；引入并发 settlement/补偿任务前；使用请求级锁或 insert-first 幂等策略让外层事务可稳定重入。
	return s.debitWithQueries(ctx, queries, params)
}

func (s *Service) debitWithQueries(ctx context.Context, queries *sqlc.Queries, params DebitParams) (Entry, error) {
	// 幂等命中表示这笔扣费已经完成，直接返回已有流水，避免重复扣余额。
	existing, err := queries.GetLedgerEntryByIdempotencyKey(ctx, params.IdempotencyKey)
	if err == nil {
		if err := ensureIdempotentEntryMatches(existing, params.UserID, params.RequestRecordID, EntryTypeDebit, params.Amount, params.Currency); err != nil {
			return Entry{}, err
		}

		return entryFromSQLC(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "lookup ledger idempotency key")
	}

	// 扣费不能自动创建余额行；不存在余额行时应视为余额不足。
	before, err := queries.GetUserBalanceForUpdate(ctx, sqlc.GetUserBalanceForUpdateParams{
		UserID:   params.UserID,
		Currency: params.Currency,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Entry{}, ledgerFailure(failure.CodeLedgerInsufficientBalance, ErrInsufficientBalance, ErrInsufficientBalance.Error())
		}
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "lock user balance")
	}

	// 让 PostgreSQL 执行 NUMERIC 减法，并通过 WHERE balance >= amount 防止扣成负数。
	after, err := queries.SubtractUserBalance(ctx, sqlc.SubtractUserBalanceParams{
		Amount:   params.Amount,
		UserID:   params.UserID,
		Currency: params.Currency,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Entry{}, ledgerFailure(failure.CodeLedgerInsufficientBalance, ErrInsufficientBalance, ErrInsufficientBalance.Error())
		}
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "subtract user balance")
	}

	created, err := queries.CreateLedgerEntry(ctx, sqlc.CreateLedgerEntryParams{
		UserID:          params.UserID,
		RequestRecordID: int64PtrToPgtypeInt8(params.RequestRecordID),
		EntryType:       string(EntryTypeDebit),
		Amount:          params.Amount,
		Currency:        params.Currency,
		BalanceBefore:   before.Balance,
		BalanceAfter:    after.Balance,
		IdempotencyKey:  params.IdempotencyKey,
		Reason:          params.Reason,
	})
	if err != nil {
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "create ledger entry")
	}

	return entryFromSQLC(created), nil
}

// resolveIdempotentCreateConflict 在并发请求同时创建同一个幂等键时返回已提交流水。
func (s *Service) resolveIdempotentCreateConflict(ctx context.Context, tx pgx.Tx, idempotencyKey string, userID int64, requestRecordID *int64, entryType EntryType, amount pgtype.Numeric, currency string) (Entry, error) {
	// PostgreSQL 写入出错后当前事务已不可继续使用，先回滚再查询已提交的幂等流水。
	_ = tx.Rollback(ctx)

	existing, err := s.queries.GetLedgerEntryByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		return Entry{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "lookup committed ledger idempotency key")
	}
	if err := ensureIdempotentEntryMatches(existing, userID, requestRecordID, entryType, amount, currency); err != nil {
		return Entry{}, err
	}

	return entryFromSQLC(existing), nil
}

// entryFromSQLC 将 sqlc 账本流水转换为 ledger 领域 DTO。
func entryFromSQLC(row sqlc.LedgerEntry) Entry {
	return Entry{
		ID:              row.ID,
		UserID:          row.UserID,
		RequestRecordID: pgtypeInt8ToInt64Ptr(row.RequestRecordID),
		EntryType:       EntryType(row.EntryType),
		Amount:          row.Amount,
		Currency:        row.Currency,
		BalanceBefore:   row.BalanceBefore,
		BalanceAfter:    row.BalanceAfter,
		IdempotencyKey:  row.IdempotencyKey,
		Reason:          row.Reason,
	}
}

// ensureIdempotentEntryMatches 校验已有幂等流水是否和本次请求语义一致。
func ensureIdempotentEntryMatches(row sqlc.LedgerEntry, userID int64, requestRecordID *int64, entryType EntryType, amount pgtype.Numeric, currency string) error {
	if row.UserID != userID {
		return ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())
	}
	if row.EntryType != string(entryType) {
		return ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())
	}
	if row.Currency != currency {
		return ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())
	}
	if !sameOptionalInt64(row.RequestRecordID, requestRecordID) {
		return ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())
	}
	if !sameNumeric(row.Amount, amount) {
		return ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())
	}

	return nil
}
