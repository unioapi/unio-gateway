package ledger

import (
	"context"
	"errors"

	"github.com/ThankCat/unio-api/internal/failure"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Reservation 表示 ledger service 返回给调用方的余额预授权记录。
type Reservation struct {
	ID                   int64
	UserID               int64
	RequestRecordID      int64
	Currency             string
	Status               ReservationStatus
	AuthorizedAmount     pgtype.Numeric
	CapturedAmount       pgtype.Numeric
	ReleasedAmount       pgtype.Numeric
	CaptureLedgerEntryID *int64
	IdempotencyKey       string
	Reason               string
}

// PreAuthorizeParams 表示余额预授权参数。
type PreAuthorizeParams struct {
	UserID          int64
	RequestRecordID int64
	Amount          pgtype.Numeric
	Currency        string
	IdempotencyKey  string
	Reason          string
}

// CaptureParams 表示预授权结算扣费参数。
type CaptureParams struct {
	RequestRecordID int64
	Amount          pgtype.Numeric
	IdempotencyKey  string
	Reason          string
}

// ReleaseParams 表示释放预授权资金参数。
type ReleaseParams struct {
	RequestRecordID int64
}

// PreAuthorize 为一次请求冻结用户余额，并创建 reservation 事实。
func (s *Service) PreAuthorize(ctx context.Context, params PreAuthorizeParams) (Reservation, error) {
	if !isPositiveNumeric(params.Amount) {
		return Reservation{}, ledgerFailure(failure.CodeLedgerInvalidAmount, ErrInvalidAmount, ErrInvalidAmount.Error())
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "begin ledger transaction")
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := s.queries.WithTx(tx)

	existing, err := txQueries.GetLedgerReservationByIdempotencyKey(ctx, params.IdempotencyKey)
	if err == nil {
		if err := ensureIdempotentReservationMatches(existing, params.UserID, params.RequestRecordID, params.Amount, params.Currency); err != nil {
			return Reservation{}, err
		}
		return reservationFromSQLC(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "lookup reservation idempotency key")
	}

	created, err := txQueries.CreateLedgerReservation(ctx, sqlc.CreateLedgerReservationParams{
		UserID:           params.UserID,
		RequestRecordID:  params.RequestRecordID,
		Currency:         params.Currency,
		AuthorizedAmount: params.Amount,
		IdempotencyKey:   params.IdempotencyKey,
		Reason:           params.Reason,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return s.resolveReservationCreateConflict(ctx, tx, params.IdempotencyKey, params.UserID, params.RequestRecordID, params.Amount, params.Currency)
		}
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "create ledger reservation")
	}

	if _, err := txQueries.ReserveUserBalance(ctx, sqlc.ReserveUserBalanceParams{
		Amount:   params.Amount,
		UserID:   params.UserID,
		Currency: params.Currency,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Reservation{}, ledgerFailure(failure.CodeLedgerInsufficientBalance, ErrInsufficientBalance, ErrInsufficientBalance.Error())
		}
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "reserve user balance")
	}

	if err := tx.Commit(ctx); err != nil {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "commit ledger transaction")
	}

	return reservationFromSQLC(created), nil
}

// Capture 将一次已预授权的请求结算为真实扣费。
// 它会在同一个事务里完成三件事：
// 1. 锁定 reservation，确认这次请求还处于 authorized 状态。
// 2. 从用户余额中扣除实际消费金额，并释放整笔预授权金额。
// 3. 写入 debit 账本流水，并把 reservation 标记为 captured。
func (s *Service) Capture(ctx context.Context, params CaptureParams) (Reservation, error) {
	// capture 表示真实扣费，金额必须大于 0。
	// 如果本次请求最终不产生费用，调用方应该走 Release，而不是用 0 金额 capture。
	if !isPositiveNumeric(params.Amount) {
		return Reservation{}, ledgerFailure(failure.CodeLedgerInvalidAmount, ErrInvalidAmount, ErrInvalidAmount.Error())
	}

	// 预授权结算会同时修改 user_balances、ledger_entries 和 ledger_reservations。
	// 这三类事实必须同生共死，所以这里显式开启事务。
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "begin ledger transaction")
	}
	// Commit 成功后 Rollback 会返回已完成事务错误，这里忽略即可。
	// 这样所有提前 return 的错误分支都能自动回滚。
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := s.queries.WithTx(tx)

	// request_record_id 在一次 gateway 请求内唯一。
	// FOR UPDATE 用来串行化同一个 request 的 capture/release 并发竞争：
	// 谁先拿到锁，谁决定 reservation 从 authorized 进入哪个终态。
	reservation, err := txQueries.GetLedgerReservationByRequestRecordIDForUpdate(ctx, params.RequestRecordID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Reservation{}, ledgerFailure(failure.CodeLedgerReservationNotFound, ErrReservationNotFound, ErrReservationNotFound.Error())
		}
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "lock ledger reservation")
	}

	switch ReservationStatus(reservation.Status) {
	case ReservationStatusCaptured:
		// 结算请求可能因为网络重试重复进入。
		// 如果已结算金额和本次金额一致，说明这是同一业务动作的幂等重放。
		if !sameNumeric(reservation.CapturedAmount, params.Amount) {
			return Reservation{}, ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())
		}

		if err := tx.Commit(ctx); err != nil {
			return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "commit ledger transaction")
		}

		return reservationFromSQLC(reservation), nil

	case ReservationStatusReleased:
		// 已释放的 reservation 不能再 capture。
		// 否则会把一次失败或取消的请求重新扣费，破坏资金语义。
		return Reservation{}, ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())

	case ReservationStatusAuthorized:
		// authorized 是唯一允许首次 capture 的状态。
		// 下面会把它推进到 captured，之后这笔 reservation 不允许再回到 authorized。

	default:
		// reservation.status 由数据库 CHECK 约束保护。
		// 走到这里通常表示代码和 schema 的状态枚举已经不一致，需要按存储错误处理。
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, errors.New("ledger: unexpected reservation status"), "unexpected reservation status")
	}

	// 实际扣费不能超过预授权金额。
	// 允许小于预授权金额：例如先按最大预算冻结，实际 usage 结算后只扣一部分。
	if !numericLessOrEqual(params.Amount, reservation.AuthorizedAmount) {
		return Reservation{}, ledgerFailure(failure.CodeLedgerInvalidAmount, ErrInvalidAmount, ErrInvalidAmount.Error())
	}

	// 锁住余额行，读取扣费前余额。
	// balance_before/balance_after 是账本流水的一部分，必须和本事务里的余额更新严格对应。
	before, err := txQueries.GetUserBalanceForUpdate(ctx, sqlc.GetUserBalanceForUpdateParams{
		UserID:   reservation.UserID,
		Currency: reservation.Currency,
	})
	if err != nil {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "lock user balance")
	}

	// 预授权结算不能复用普通 Debit：
	// Debit 只减少 balance，而 Capture 必须同时减少 balance 和 reserved_balance。
	// 这里扣除的 balance 是实际消费金额，释放的 reserved_balance 是整笔预授权金额。
	// 如果实际消费小于预授权金额，差额会通过 released_amount 记录在 reservation 上。
	after, err := txQueries.CaptureUserReservedBalance(ctx, sqlc.CaptureUserReservedBalanceParams{
		CapturedAmount:   params.Amount,
		AuthorizedAmount: reservation.AuthorizedAmount,
		UserID:           reservation.UserID,
		Currency:         reservation.Currency,
	})
	if err != nil {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "capture user reserved balance")
	}

	requestRecordID := reservation.RequestRecordID
	// 真实扣费必须写 ledger entry，因为 balance 发生了变化。
	// 这条 debit 流水是后续审计、用户账单和 request log 对账的核心事实。
	entry, err := txQueries.CreateLedgerEntry(ctx, sqlc.CreateLedgerEntryParams{
		UserID:          reservation.UserID,
		RequestRecordID: int64PtrToPgtypeInt8(&requestRecordID),
		EntryType:       string(EntryTypeDebit),
		Amount:          params.Amount,
		Currency:        reservation.Currency,
		BalanceBefore:   before.Balance,
		BalanceAfter:    after.Balance,
		IdempotencyKey:  params.IdempotencyKey,
		Reason:          params.Reason,
	})
	if err != nil {
		// reservation 行已经被 FOR UPDATE 锁住。
		// 正常并发重试会在锁释放后看到 captured 状态，不会走到这里。
		// 因此这里的唯一冲突通常表示幂等键被其他业务请求复用。
		if isUniqueViolation(err) {
			return Reservation{}, ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())
		}

		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "create capture ledger entry")
	}

	// 最后再更新 reservation 终态，并把 capture_ledger_entry_id 指向刚创建的 debit 流水。
	// 这样 reservation 和 ledger entry 之间形成可追溯关系：
	// request -> reservation -> debit ledger entry。
	captured, err := txQueries.CaptureLedgerReservation(ctx, sqlc.CaptureLedgerReservationParams{
		CapturedAmount: params.Amount,
		CaptureLedgerEntryID: pgtype.Int8{
			Int64: entry.ID,
			Valid: true,
		},
		ID: reservation.ID,
	})
	if err != nil {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "capture ledger reservation")
	}

	// 只有余额、流水、reservation 三者都更新成功，才提交本次 capture。
	if err := tx.Commit(ctx); err != nil {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "commit ledger transaction")
	}

	return reservationFromSQLC(captured), nil
}

// Release 释放一次尚未结算的预授权金额。
// 它只减少 reserved_balance，不写 ledger entry，因为用户真实余额 balance 没有变化。
func (s *Service) Release(ctx context.Context, params ReleaseParams) (Reservation, error) {
	// release 会同时修改 user_balances.reserved_balance 和 ledger_reservations。
	// 两者必须在同一个事务里完成，避免出现 reservation 已释放但冻结余额还在的状态。
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "begin ledger transaction")
	}
	// Commit 成功后的 Rollback 会被忽略；错误路径自动回滚。
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := s.queries.WithTx(tx)

	// 和 Capture 一样，Release 也按 request_record_id 锁 reservation。
	// 这可以保证同一个请求的 capture/release 并发到达时，只有一个终态会成功落库。
	reservation, err := txQueries.GetLedgerReservationByRequestRecordIDForUpdate(ctx, params.RequestRecordID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Reservation{}, ledgerFailure(failure.CodeLedgerReservationNotFound, ErrReservationNotFound, ErrReservationNotFound.Error())
		}
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "lock ledger reservation")
	}

	switch ReservationStatus(reservation.Status) {
	case ReservationStatusReleased:
		// release 是按 request_record_id 做幂等的。
		// 重试释放同一笔 reservation，不应该重复减少 reserved_balance。
		if err := tx.Commit(ctx); err != nil {
			return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "commit ledger transaction")
		}

		return reservationFromSQLC(reservation), nil

	case ReservationStatusCaptured:
		// 已经结算扣费的 reservation 不能再释放。
		// 释放 captured reservation 会让 reserved_balance 和真实扣费事实不一致。
		return Reservation{}, ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())

	case ReservationStatusAuthorized:
		// authorized 是唯一允许首次 release 的状态。
		// 下面只释放冻结金额，不产生扣费流水。

	default:
		// reservation.status 由数据库 CHECK 约束保护。
		// 这里兜底处理 schema 和代码状态枚举不一致的异常情况。
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, errors.New("ledger: unexpected reservation status"), "unexpected reservation status")
	}

	// Release 只减少 reserved_balance，不减少 balance。
	// 因为这表示请求没有产生真实费用，或者请求失败后需要解除冻结。
	_, err = txQueries.ReleaseUserReservedBalance(ctx, sqlc.ReleaseUserReservedBalanceParams{
		Amount:   reservation.AuthorizedAmount,
		UserID:   reservation.UserID,
		Currency: reservation.Currency,
	})
	if err != nil {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "release user reserved balance")
	}

	// 把 reservation 推进到 released 终态。
	// released_amount 等于 authorized_amount，表示整笔冻结金额已经释放。
	released, err := txQueries.ReleaseLedgerReservation(ctx, reservation.ID)
	if err != nil {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "release ledger reservation")
	}

	// 余额冻结释放和 reservation 终态都成功后，才提交本次 release。
	if err := tx.Commit(ctx); err != nil {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "commit ledger transaction")
	}

	return reservationFromSQLC(released), nil
}

// resolveReservationCreateConflict 在 reservation 唯一约束冲突后解析冲突来源。
func (s *Service) resolveReservationCreateConflict(ctx context.Context, tx pgx.Tx, idempotencyKey string, userID int64, requestRecordID int64, amount pgtype.Numeric, currency string) (Reservation, error) {
	_ = tx.Rollback(ctx)

	// 第一优先级查 idempotency_key：同一个幂等键并发进入时，应返回已有 reservation，视为幂等成功。
	existing, err := s.queries.GetLedgerReservationByIdempotencyKey(ctx, idempotencyKey)
	if err == nil {
		if err := ensureIdempotentReservationMatches(existing, userID, requestRecordID, amount, currency); err != nil {
			return Reservation{}, err
		}
		return reservationFromSQLC(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "lookup committed reservation idempotency key")
	}

	// 第二优先级查 request_record_id：同一个 request 被不同幂等键重复预授权，应视为业务幂等冲突。
	existing, err = s.queries.GetLedgerReservationByRequestRecordID(ctx, requestRecordID)
	if err == nil {
		return Reservation{}, ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, err, "lookup committed reservation request")
	}

	// 理论上唯一冲突只能来自 idempotency_key 或 request_record_id；两个都查不到说明状态不符合当前业务假设。
	return Reservation{}, ledgerFailure(failure.CodeLedgerStoreFailed, pgx.ErrNoRows, "resolve reservation create conflict")
}

// reservationFromSQLC 将 sqlc 预扣除记录转换为 ledger 领域 DTO。
func reservationFromSQLC(row sqlc.LedgerReservation) Reservation {
	return Reservation{
		ID:                   row.ID,
		UserID:               row.UserID,
		RequestRecordID:      row.RequestRecordID,
		Currency:             row.Currency,
		Status:               ReservationStatus(row.Status),
		AuthorizedAmount:     row.AuthorizedAmount,
		CapturedAmount:       row.CapturedAmount,
		ReleasedAmount:       row.ReleasedAmount,
		CaptureLedgerEntryID: pgtypeInt8ToInt64Ptr(row.CaptureLedgerEntryID),
		IdempotencyKey:       row.IdempotencyKey,
		Reason:               row.Reason,
	}
}

func ensureIdempotentReservationMatches(row sqlc.LedgerReservation, userID int64, requestRecordID int64, amount pgtype.Numeric, currency string) error {
	if row.UserID != userID {
		return ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())
	}
	if row.RequestRecordID != requestRecordID {
		return ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())
	}
	if row.Currency != currency {
		return ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())
	}
	if !sameNumeric(row.AuthorizedAmount, amount) {
		return ledgerFailure(failure.CodeLedgerIdempotencyConflict, ErrIdempotencyConflict, ErrIdempotencyConflict.Error())
	}

	return nil
}
