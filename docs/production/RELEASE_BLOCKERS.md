# Release Blockers

本文档只记录公开生产前必须解决的阻断项。

## 当前阻断项

| ID | GAP | 阶段 | 阻断原因 | 关联任务 |
| --- | --- | --- | --- | --- |
| RB-005 | [GAP-7-003](TODO_REGISTER.md#gap-7-003) | 阶段 7 | request/attempt 终态缺少状态机守卫，并发更新可能覆盖账务事实。 | [TASK-7.18](../chapters/phase-07-billing-ledger/PLAN.md#task-7-18-request-state-machine) |
| RB-007 | [GAP-7-007](TODO_REGISTER.md#gap-7-007) | 阶段 7 | settlement 缺少请求级幂等完成检测；上游成功后的 settlement 失败可能导致冻结余额悬挂，后续需由 worker recovery 持久化补偿并幂等重试。 | [TASK-7.19](../chapters/phase-07-billing-ledger/PLAN.md#task-7-19-settlement-idempotency) |
| RB-009 | [GAP-7-012](TODO_REGISTER.md#gap-7-012) | 阶段 7 | 外部事务内并发 debit 幂等冲突可能导致 settlement 失败且无法稳定重入，worker recovery 前必须解决。 | [TASK-7.19](../chapters/phase-07-billing-ledger/PLAN.md#task-7-19-settlement-idempotency) |

## 使用规则

1. 任何 `P0` 且 `release_blocker=yes` 的 GAP 必须同步进入本文档。
2. blocker 关闭时，先完成代码和测试，再更新 TODO register，最后移出本文档。
3. 本文档不记录普通优化项，只记录影响公开生产、资金、安全、账务或用户契约的阻断项。
