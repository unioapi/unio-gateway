# Release Blockers

本文档只记录公开生产前必须解决的阻断项。

## 当前阻断项

| ID | GAP | 阶段 | 阻断原因 | 关联任务 |
| --- | --- | --- | --- | --- |
| RB-007 | [GAP-7-007](TODO_REGISTER.md#gap-7-007) | 阶段 7 | settlement 已支持成功重放一致性检查；上游成功后的首次 settlement 失败仍可能导致冻结余额悬挂，后续需由 worker recovery 持久化补偿并幂等重试。 | [TASK-7.19](../chapters/phase-07-billing-ledger/PLAN.md#task-7-19-settlement-idempotency) |

## 使用规则

1. 任何 `P0` 且 `release_blocker=yes` 的 GAP 必须同步进入本文档。
2. blocker 关闭时，先完成代码和测试，再更新 TODO register，最后移出本文档。
3. 本文档不记录普通优化项，只记录影响公开生产、资金、安全、账务或用户契约的阻断项。
