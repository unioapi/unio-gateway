package ledger

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ThankCat/unio-api/internal/store/sqlc"
)

var (
	// ErrInsufficientBalance 表示用户余额不足，不能完成扣费。
	ErrInsufficientBalance = errors.New("ledger: insufficient balance")
	// ErrIdempotencyConflict 表示同一个幂等键被不同账本参数复用。
	ErrIdempotencyConflict = errors.New("ledger: idempotency key conflict")
)

// TxBeginner 定义 ledger service 开启数据库事务所需能力。
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Service 负责用户余额变动和账本流水写入。
type Service struct {
	db      TxBeginner
	queries *sqlc.Queries
}

// Entry 表示 ledger service 返回给调用方的账本流水。
type Entry struct {
	ID              int64
	UserID          int64
	RequestRecordID *int64
	EntryType       string
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

// NewService 创建 ledger service。
func NewService(db TxBeginner, queries *sqlc.Queries) *Service {
	return &Service{db: db, queries: queries}
}

// Credit 增加用户余额，并在同一个事务里写入 credit 账本流水。
func (s *Service) Credit(ctx context.Context, params CreditParams) (Entry, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Entry{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := s.queries.WithTx(tx)

	// 幂等命中表示这笔加款已经完成，直接返回已有流水，避免重复加余额。
	existing, err := txQueries.GetLedgerEntryByIdempotencyKey(ctx, params.IdempotencyKey)
	if err == nil {
		if err := ensureIdempotentEntryMatches(existing, params.UserID, params.RequestRecordID, "credit", params.Amount, params.Currency); err != nil {
			return Entry{}, err
		}

		return entryFromSQLC(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, err
	}

	// Credit 可以为新用户创建 0 余额行，再在同一事务中加款。
	if err := txQueries.EnsureUserBalance(ctx, sqlc.EnsureUserBalanceParams{
		UserID:   params.UserID,
		Currency: params.Currency,
	}); err != nil {
		return Entry{}, err
	}

	// 锁定用户余额行，确保并发充值/扣费不会基于同一个旧余额计算。
	before, err := txQueries.GetUserBalanceForUpdate(ctx, sqlc.GetUserBalanceForUpdateParams{
		UserID:   params.UserID,
		Currency: params.Currency,
	})
	if err != nil {
		return Entry{}, err
	}

	// 让 PostgreSQL 执行 NUMERIC 加法，避免在 Go 中用 float 或手写 decimal 计算金额。
	after, err := txQueries.AddUserBalance(ctx, sqlc.AddUserBalanceParams{
		Amount:   params.Amount,
		UserID:   params.UserID,
		Currency: params.Currency,
	})
	if err != nil {
		return Entry{}, err
	}

	// 写入账本事实；balance_before/after 必须和余额更新结果一致。
	created, err := txQueries.CreateLedgerEntry(ctx, sqlc.CreateLedgerEntryParams{
		UserID:          params.UserID,
		RequestRecordID: int64PtrToPgtypeInt8(params.RequestRecordID),
		EntryType:       "credit",
		Amount:          params.Amount,
		Currency:        params.Currency,
		BalanceBefore:   before.Balance,
		BalanceAfter:    after.Balance,
		IdempotencyKey:  params.IdempotencyKey,
		Reason:          params.Reason,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return s.resolveIdempotentCreateConflict(ctx, tx, params.IdempotencyKey, params.UserID, params.RequestRecordID, "credit", params.Amount, params.Currency)
		}

		return Entry{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Entry{}, err
	}

	return entryFromSQLC(created), nil
}

// Debit 减少用户余额，并在同一个事务里写入 debit 账本流水。
func (s *Service) Debit(ctx context.Context, params DebitParams) (Entry, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Entry{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := s.queries.WithTx(tx)

	// 幂等命中表示这笔扣费已经完成，直接返回已有流水，避免重复扣余额。
	existing, err := txQueries.GetLedgerEntryByIdempotencyKey(ctx, params.IdempotencyKey)
	if err == nil {
		if err := ensureIdempotentEntryMatches(existing, params.UserID, params.RequestRecordID, "debit", params.Amount, params.Currency); err != nil {
			return Entry{}, err
		}

		return entryFromSQLC(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, err
	}

	// 扣费不能自动创建余额行；不存在余额行时应视为余额不足。
	before, err := txQueries.GetUserBalanceForUpdate(ctx, sqlc.GetUserBalanceForUpdateParams{
		UserID:   params.UserID,
		Currency: params.Currency,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Entry{}, ErrInsufficientBalance
		}
		return Entry{}, err
	}

	// 让 PostgreSQL 执行 NUMERIC 减法，并通过 WHERE balance >= amount 防止扣成负数。
	after, err := txQueries.SubtractUserBalance(ctx, sqlc.SubtractUserBalanceParams{
		Amount:   params.Amount,
		UserID:   params.UserID,
		Currency: params.Currency,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Entry{}, ErrInsufficientBalance
		}
		return Entry{}, err
	}

	// 写入账本事实；debit 的 balance_after 必须等于 balance_before - amount。
	created, err := txQueries.CreateLedgerEntry(ctx, sqlc.CreateLedgerEntryParams{
		UserID:          params.UserID,
		RequestRecordID: int64PtrToPgtypeInt8(params.RequestRecordID),
		EntryType:       "debit",
		Amount:          params.Amount,
		Currency:        params.Currency,
		BalanceBefore:   before.Balance,
		BalanceAfter:    after.Balance,
		IdempotencyKey:  params.IdempotencyKey,
		Reason:          params.Reason,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return s.resolveIdempotentCreateConflict(ctx, tx, params.IdempotencyKey, params.UserID, params.RequestRecordID, "debit", params.Amount, params.Currency)
		}

		return Entry{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Entry{}, err
	}

	return entryFromSQLC(created), nil
}

// resolveIdempotentCreateConflict 在并发请求同时创建同一个幂等键时返回已提交流水。
func (s *Service) resolveIdempotentCreateConflict(ctx context.Context, tx pgx.Tx, idempotencyKey string, userID int64, requestRecordID *int64, entryType string, amount pgtype.Numeric, currency string) (Entry, error) {
	// PostgreSQL 写入出错后当前事务已不可继续使用，先回滚再查询已提交的幂等流水。
	_ = tx.Rollback(ctx)

	existing, err := s.queries.GetLedgerEntryByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		return Entry{}, err
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
		EntryType:       row.EntryType,
		Amount:          row.Amount,
		Currency:        row.Currency,
		BalanceBefore:   row.BalanceBefore,
		BalanceAfter:    row.BalanceAfter,
		IdempotencyKey:  row.IdempotencyKey,
		Reason:          row.Reason,
	}
}

// pgtypeInt8ToInt64Ptr 将 pgtype.Int8 转成可选 int64 指针。
func pgtypeInt8ToInt64Ptr(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}

	return &value.Int64
}

// int64PtrToPgtypeInt8 将可选 int64 指针转成 pgtype.Int8。
func int64PtrToPgtypeInt8(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{Valid: false}
	}

	return pgtype.Int8{Int64: *value, Valid: true}
}

// isUniqueViolation 判断数据库错误是否是唯一约束冲突。
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// ensureIdempotentEntryMatches 校验已有幂等流水是否和本次请求语义一致。
func ensureIdempotentEntryMatches(row sqlc.LedgerEntry, userID int64, requestRecordID *int64, entryType string, amount pgtype.Numeric, currency string) error {
	if row.UserID != userID {
		return ErrIdempotencyConflict
	}
	if row.EntryType != entryType {
		return ErrIdempotencyConflict
	}
	if row.Currency != currency {
		return ErrIdempotencyConflict
	}
	if !sameOptionalInt64(row.RequestRecordID, requestRecordID) {
		return ErrIdempotencyConflict
	}
	if !sameNumeric(row.Amount, amount) {
		return ErrIdempotencyConflict
	}

	return nil
}

// sameOptionalInt64 比较数据库可空 int8 和领域层可选 int64 是否相同。
func sameOptionalInt64(left pgtype.Int8, right *int64) bool {
	if !left.Valid {
		return right == nil
	}
	if right == nil {
		return false
	}

	return left.Int64 == *right
}

// sameNumeric 比较两个 pgtype.Numeric 是否表示同一个测试期金额值。
func sameNumeric(left pgtype.Numeric, right pgtype.Numeric) bool {
	if left.Valid != right.Valid {
		return false
	}
	if !left.Valid {
		return true
	}
	if left.Exp != right.Exp {
		return false
	}
	if left.Int == nil || right.Int == nil {
		return left.Int == right.Int
	}

	return left.Int.Cmp(right.Int) == 0
}
