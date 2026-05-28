package workers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ThankCat/unio-api/internal/platform/failure"
	"github.com/ThankCat/unio-api/internal/platform/store/sqlc"
)

type fakeSettlementRecoveryJobStore struct {
	claimJob       sqlc.SettlementRecoveryJob
	claimErr       error
	exhaustedJob   sqlc.SettlementRecoveryJob
	exhaustedErr   error
	exhaustedFound bool

	claimArgs     []sqlc.ClaimNextSettlementRecoveryJobParams
	succeededArgs []sqlc.MarkSettlementRecoveryJobSucceededParams
	retryArgs     []sqlc.MarkSettlementRecoveryJobRetryParams
	deadArgs      []sqlc.MarkSettlementRecoveryJobDeadParams
	exhaustedArgs []sqlc.MarkExhaustedSettlementRecoveryJobDeadParams
}

func (s *fakeSettlementRecoveryJobStore) ClaimNextSettlementRecoveryJob(ctx context.Context, arg sqlc.ClaimNextSettlementRecoveryJobParams) (sqlc.SettlementRecoveryJob, error) {
	s.claimArgs = append(s.claimArgs, arg)
	if s.claimErr != nil {
		return sqlc.SettlementRecoveryJob{}, s.claimErr
	}
	if s.claimJob.ID == 0 {
		return sqlc.SettlementRecoveryJob{}, pgx.ErrNoRows
	}
	return s.claimJob, nil
}

func (s *fakeSettlementRecoveryJobStore) MarkSettlementRecoveryJobSucceeded(ctx context.Context, arg sqlc.MarkSettlementRecoveryJobSucceededParams) (sqlc.MarkSettlementRecoveryJobSucceededRow, error) {
	s.succeededArgs = append(s.succeededArgs, arg)
	return sqlc.MarkSettlementRecoveryJobSucceededRow{ID: arg.ID, Status: "succeeded"}, nil
}

func (s *fakeSettlementRecoveryJobStore) MarkSettlementRecoveryJobRetry(ctx context.Context, arg sqlc.MarkSettlementRecoveryJobRetryParams) (sqlc.SettlementRecoveryJob, error) {
	s.retryArgs = append(s.retryArgs, arg)
	return sqlc.SettlementRecoveryJob{ID: arg.ID, Status: "pending"}, nil
}

func (s *fakeSettlementRecoveryJobStore) MarkSettlementRecoveryJobDead(ctx context.Context, arg sqlc.MarkSettlementRecoveryJobDeadParams) (sqlc.SettlementRecoveryJob, error) {
	s.deadArgs = append(s.deadArgs, arg)
	return sqlc.SettlementRecoveryJob{ID: arg.ID, Status: "dead"}, nil
}

func (s *fakeSettlementRecoveryJobStore) MarkExhaustedSettlementRecoveryJobDead(ctx context.Context, arg sqlc.MarkExhaustedSettlementRecoveryJobDeadParams) (sqlc.SettlementRecoveryJob, error) {
	s.exhaustedArgs = append(s.exhaustedArgs, arg)
	if s.exhaustedErr != nil {
		return sqlc.SettlementRecoveryJob{}, s.exhaustedErr
	}
	if !s.exhaustedFound {
		return sqlc.SettlementRecoveryJob{}, pgx.ErrNoRows
	}
	return s.exhaustedJob, nil
}

type fakeSettlementRecoveryRecoverer struct {
	jobs []sqlc.SettlementRecoveryJob
	err  error
}

func (r *fakeSettlementRecoveryRecoverer) RecoverChatSettlement(ctx context.Context, job sqlc.SettlementRecoveryJob) error {
	r.jobs = append(r.jobs, job)
	return r.err
}

func TestSettlementRecoveryWorkerRunOnceReturnsIdleWhenNoJob(t *testing.T) {
	store := &fakeSettlementRecoveryJobStore{}
	worker := NewSettlementRecoveryWorker(store, &fakeSettlementRecoveryRecoverer{}, "worker-a", time.Second)

	worked, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned err: %v", err)
	}
	if worked {
		t.Fatal("expected idle worker when no job exists")
	}
	if len(store.claimArgs) != 1 {
		t.Fatalf("expected one claim attempt, got %d", len(store.claimArgs))
	}
	if store.claimArgs[0].LockedBy.String != "worker-a" || !store.claimArgs[0].LockedBy.Valid {
		t.Fatalf("expected locked_by worker-a, got %#v", store.claimArgs[0].LockedBy)
	}
}

func TestSettlementRecoveryWorkerMarksSucceededAfterRecovery(t *testing.T) {
	store := &fakeSettlementRecoveryJobStore{claimJob: recoveryJob(10, 1, 3)}
	recoverer := &fakeSettlementRecoveryRecoverer{}
	worker := NewSettlementRecoveryWorker(store, recoverer, "worker-a", time.Second)

	worked, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned err: %v", err)
	}
	if !worked {
		t.Fatal("expected worker to process job")
	}
	if len(recoverer.jobs) != 1 || recoverer.jobs[0].ID != 10 {
		t.Fatalf("expected recoverer to receive job 10, got %#v", recoverer.jobs)
	}
	if len(store.succeededArgs) != 1 || store.succeededArgs[0].ID != 10 {
		t.Fatalf("expected job 10 to be marked succeeded, got %#v", store.succeededArgs)
	}
	if len(store.retryArgs) != 0 || len(store.deadArgs) != 0 {
		t.Fatalf("expected no retry/dead mark, got retry=%d dead=%d", len(store.retryArgs), len(store.deadArgs))
	}
}

func TestSettlementRecoveryWorkerRetriesRecoverableFailure(t *testing.T) {
	recoverErr := failure.New(failure.CodeGatewayChatSettlementFailed, failure.WithMessage("temporary settlement failure"))
	store := &fakeSettlementRecoveryJobStore{claimJob: recoveryJob(20, 2, 4)}
	worker := NewSettlementRecoveryWorker(store, &fakeSettlementRecoveryRecoverer{err: recoverErr}, "worker-a", time.Second)

	worked, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned err: %v", err)
	}
	if !worked {
		t.Fatal("expected worker to process failed job")
	}
	if len(store.retryArgs) != 1 || store.retryArgs[0].ID != 20 {
		t.Fatalf("expected job 20 to be marked retry, got %#v", store.retryArgs)
	}
	if store.retryArgs[0].LastErrorCode.String != string(failure.CodeGatewayChatSettlementFailed) {
		t.Fatalf("expected failure code, got %#v", store.retryArgs[0].LastErrorCode)
	}
	if !store.retryArgs[0].NextRunAt.Valid || !store.retryArgs[0].NextRunAt.Time.After(time.Now()) {
		t.Fatalf("expected future retry time, got %#v", store.retryArgs[0].NextRunAt)
	}
	if len(store.deadArgs) != 0 {
		t.Fatalf("expected no dead mark before max attempts, got %d", len(store.deadArgs))
	}
}

func TestSettlementRecoveryWorkerMarksDeadOnFinalFailure(t *testing.T) {
	store := &fakeSettlementRecoveryJobStore{claimJob: recoveryJob(30, 3, 3)}
	worker := NewSettlementRecoveryWorker(store, &fakeSettlementRecoveryRecoverer{err: errors.New("permanent settlement failure")}, "worker-a", time.Second)

	worked, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned err: %v", err)
	}
	if !worked {
		t.Fatal("expected worker to process final failed job")
	}
	if len(store.deadArgs) != 1 || store.deadArgs[0].ID != 30 {
		t.Fatalf("expected job 30 to be marked dead, got %#v", store.deadArgs)
	}
	if store.deadArgs[0].LastErrorCode.String != string(failure.CodeGatewayChatSettlementFailed) {
		t.Fatalf("expected fallback failure code, got %#v", store.deadArgs[0].LastErrorCode)
	}
	if len(store.retryArgs) != 0 {
		t.Fatalf("expected no retry at max attempts, got %d", len(store.retryArgs))
	}
}

func TestSettlementRecoveryWorkerMarksExhaustedJobDeadBeforeClaim(t *testing.T) {
	store := &fakeSettlementRecoveryJobStore{
		exhaustedFound: true,
		exhaustedJob:   recoveryJob(40, 3, 3),
		claimJob:       recoveryJob(50, 1, 3),
	}
	recoverer := &fakeSettlementRecoveryRecoverer{}
	worker := NewSettlementRecoveryWorker(store, recoverer, "worker-a", time.Second)

	worked, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned err: %v", err)
	}
	if !worked {
		t.Fatal("expected exhausted job to count as work")
	}
	if len(store.exhaustedArgs) != 1 {
		t.Fatalf("expected one exhausted mark, got %d", len(store.exhaustedArgs))
	}
	if !store.exhaustedArgs[0].CompletedAt.Valid {
		t.Fatal("expected exhausted dead mark to set completed_at")
	}
	if len(store.claimArgs) != 0 {
		t.Fatalf("expected claim to be skipped after exhausted mark, got %d", len(store.claimArgs))
	}
	if len(recoverer.jobs) != 0 {
		t.Fatalf("expected recoverer not to run for exhausted job, got %d calls", len(recoverer.jobs))
	}
}

func recoveryJob(id int64, attemptCount int32, maxAttempts int32) sqlc.SettlementRecoveryJob {
	return sqlc.SettlementRecoveryJob{
		ID:           id,
		Status:       "running",
		AttemptCount: attemptCount,
		MaxAttempts:  maxAttempts,
		LockedBy:     pgtype.Text{String: "worker-a", Valid: true},
		LockedUntil:  pgtype.Timestamptz{Time: time.Now().Add(time.Second), Valid: true},
	}
}
