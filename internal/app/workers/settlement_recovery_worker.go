package workers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ThankCat/unio-api/internal/platform/failure"
	"github.com/ThankCat/unio-api/internal/platform/store/sqlc"
)

const (
	defaultSettlementRecoveryLockTTL = 30 * time.Second
	maxRecoveryErrorDetailBytes      = 2048
)

// SettlementRecoveryJobStore 定义 worker claim 和推进 recovery job 状态所需的存储能力。
type SettlementRecoveryJobStore interface {
	ClaimNextSettlementRecoveryJob(ctx context.Context, arg sqlc.ClaimNextSettlementRecoveryJobParams) (sqlc.SettlementRecoveryJob, error)
	MarkSettlementRecoveryJobSucceeded(ctx context.Context, arg sqlc.MarkSettlementRecoveryJobSucceededParams) (sqlc.MarkSettlementRecoveryJobSucceededRow, error)
	MarkSettlementRecoveryJobRetry(ctx context.Context, arg sqlc.MarkSettlementRecoveryJobRetryParams) (sqlc.SettlementRecoveryJob, error)
	MarkSettlementRecoveryJobDead(ctx context.Context, arg sqlc.MarkSettlementRecoveryJobDeadParams) (sqlc.SettlementRecoveryJob, error)
	MarkExhaustedSettlementRecoveryJobDead(ctx context.Context, arg sqlc.MarkExhaustedSettlementRecoveryJobDeadParams) (sqlc.SettlementRecoveryJob, error)
}

// SettlementRecoveryRecoverer 定义 worker 重放 settlement recovery job 的业务能力。
type SettlementRecoveryRecoverer interface {
	RecoverChatSettlement(ctx context.Context, job sqlc.SettlementRecoveryJob) error
}

// SettlementRecoveryWorker claim settlement_recovery_jobs 并驱动幂等 settlement 重试。
type SettlementRecoveryWorker struct {
	store     SettlementRecoveryJobStore
	recoverer SettlementRecoveryRecoverer
	workerID  string
	lockTTL   time.Duration
}

// NewSettlementRecoveryWorker 创建 settlement recovery worker。
func NewSettlementRecoveryWorker(store SettlementRecoveryJobStore, recoverer SettlementRecoveryRecoverer, workerID string, lockTTL time.Duration) *SettlementRecoveryWorker {
	if store == nil {
		panic("workers: settlement recovery store is required")
	}
	if recoverer == nil {
		panic("workers: settlement recovery recoverer is required")
	}
	if workerID == "" {
		workerID = "settlement-recovery-worker"
	}
	if lockTTL <= 0 {
		lockTTL = defaultSettlementRecoveryLockTTL
	}

	return &SettlementRecoveryWorker{
		store:     store,
		recoverer: recoverer,
		workerID:  workerID,
		lockTTL:   lockTTL,
	}
}

// Name 返回 worker 名称。
func (w *SettlementRecoveryWorker) Name() string {
	return "settlement_recovery"
}

// RunOnce claim 并处理一条到期 settlement recovery job。
func (w *SettlementRecoveryWorker) RunOnce(ctx context.Context) (bool, error) {
	now := time.Now()
	if worked, err := w.markExhausted(ctx, now); worked || err != nil {
		return worked, err
	}

	job, err := w.store.ClaimNextSettlementRecoveryJob(ctx, sqlc.ClaimNextSettlementRecoveryJobParams{
		LockedBy: pgtype.Text{String: w.workerID, Valid: true},
		LockedUntil: pgtype.Timestamptz{
			Time:  now.Add(w.lockTTL),
			Valid: true,
		},
		NowAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, failure.Wrap(
			failure.CodeGatewayChatSettlementFailed,
			err,
			failure.WithMessage("claim settlement recovery job"),
		)
	}

	jobCtx, cancel := context.WithTimeout(ctx, w.lockTTL)
	defer cancel()

	if err := w.recoverer.RecoverChatSettlement(jobCtx, job); err != nil {
		return true, w.markFailed(ctx, job, err)
	}

	_, err = w.store.MarkSettlementRecoveryJobSucceeded(ctx, sqlc.MarkSettlementRecoveryJobSucceededParams{
		ID:          job.ID,
		CompletedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return true, failure.Wrap(
			failure.CodeGatewayChatSettlementFailed,
			err,
			failure.WithMessage("mark settlement recovery job succeeded"),
		)
	}

	return true, nil
}

func (w *SettlementRecoveryWorker) markExhausted(ctx context.Context, now time.Time) (bool, error) {
	_, err := w.store.MarkExhaustedSettlementRecoveryJobDead(ctx, sqlc.MarkExhaustedSettlementRecoveryJobDeadParams{
		NowAt:                   pgtype.Timestamptz{Time: now, Valid: true},
		LastErrorCode:           nullableText(string(failure.CodeGatewayChatSettlementFailed)),
		LastErrorMessage:        nullableText("Settlement recovery attempts exhausted."),
		LastInternalErrorDetail: nullableText("settlement recovery job reached max_attempts before worker could claim another retry"),
		CompletedAt:             pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, failure.Wrap(
			failure.CodeGatewayChatSettlementFailed,
			err,
			failure.WithMessage("mark exhausted settlement recovery job dead"),
		)
	}

	return true, nil
}

func (w *SettlementRecoveryWorker) markFailed(ctx context.Context, job sqlc.SettlementRecoveryJob, recoverErr error) error {
	now := time.Now()
	code, safeMessage, internalDetail := recoveryErrorFacts(recoverErr)

	if job.AttemptCount >= job.MaxAttempts {
		_, err := w.store.MarkSettlementRecoveryJobDead(ctx, sqlc.MarkSettlementRecoveryJobDeadParams{
			ID:                      job.ID,
			LastErrorCode:           nullableText(code),
			LastErrorMessage:        nullableText(safeMessage),
			LastInternalErrorDetail: nullableText(internalDetail),
			CompletedAt:             pgtype.Timestamptz{Time: now, Valid: true},
		})
		if err != nil {
			return failure.Wrap(
				failure.CodeGatewayChatSettlementFailed,
				err,
				failure.WithMessage("mark settlement recovery job dead"),
			)
		}

		return nil
	}

	_, err := w.store.MarkSettlementRecoveryJobRetry(ctx, sqlc.MarkSettlementRecoveryJobRetryParams{
		ID:                      job.ID,
		NextRunAt:               pgtype.Timestamptz{Time: now.Add(settlementRecoveryBackoff(job.AttemptCount)), Valid: true},
		LastErrorCode:           nullableText(code),
		LastErrorMessage:        nullableText(safeMessage),
		LastInternalErrorDetail: nullableText(internalDetail),
		UpdatedAt:               pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return failure.Wrap(
			failure.CodeGatewayChatSettlementFailed,
			err,
			failure.WithMessage("mark settlement recovery job retry"),
		)
	}

	return nil
}

func settlementRecoveryBackoff(attemptCount int32) time.Duration {
	if attemptCount <= 1 {
		return time.Second
	}

	shift := attemptCount - 1
	if shift > 6 {
		shift = 6
	}

	return time.Second * time.Duration(1<<shift)
}

func recoveryErrorFacts(err error) (code string, safeMessage string, internalDetail string) {
	code = string(failure.CodeOf(err))
	if code == "" {
		code = string(failure.CodeGatewayChatSettlementFailed)
	}

	return code, "Settlement recovery failed.", truncateRecoveryErrorDetail(err)
}

func truncateRecoveryErrorDetail(err error) string {
	if err == nil {
		return ""
	}

	detail := strings.TrimSpace(err.Error())
	if len(detail) <= maxRecoveryErrorDetailBytes {
		return detail
	}

	return detail[:maxRecoveryErrorDetailBytes] + "...[truncated]"
}

func nullableText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{Valid: false}
	}

	return pgtype.Text{String: value, Valid: true}
}
