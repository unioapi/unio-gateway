package workers

import (
	"context"
	"log/slog"
	"time"

	"github.com/ThankCat/unio-api/internal/platform/failure"
)

const defaultRunnerIdleInterval = time.Second

// Unit 表示 runner 可调度的一类后台任务。
type Unit interface {
	// Name 返回 worker 名称，用于日志和 worker id。
	Name() string
	// RunOnce 执行一次工作；返回 true 表示本轮处理了任务。
	RunOnce(ctx context.Context) (bool, error)
}

// Runner 顺序调度一组后台 worker，并在空闲时按固定间隔休眠。
type Runner struct {
	logger       *slog.Logger
	idleInterval time.Duration
	workers      []Unit
}

// NewRunner 创建后台 worker runner。
func NewRunner(logger *slog.Logger, idleInterval time.Duration, units ...Unit) *Runner {
	if logger == nil {
		logger = slog.Default()
	}
	if idleInterval <= 0 {
		idleInterval = defaultRunnerIdleInterval
	}

	return &Runner{
		logger:       logger,
		idleInterval: idleInterval,
		workers:      units,
	}
}

// Run 持续执行 worker，直到 ctx 被取消。
func (r *Runner) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		worked := false
		for _, unit := range r.workers {
			if unit == nil {
				continue
			}

			didWork, err := unit.RunOnce(ctx)
			if err != nil {
				args := append([]any{"worker", unit.Name()}, failure.LogArgs(err)...)
				r.logger.Error("worker run failed", args...)
			}
			worked = worked || didWork
		}

		if worked {
			continue
		}

		timer := time.NewTimer(r.idleInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		case <-timer.C:
		}
	}
}
