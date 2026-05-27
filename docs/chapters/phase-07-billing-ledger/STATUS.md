# Phase 7 Status

状态：in_progress

## 已完成

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.01 | partial | 客户侧 price schema 已建立，成本价和生效窗口约束未完成。 |
| TASK-7.02 | done | request/attempt 记录、状态机守卫、safe error message 和 internal error detail 审计字段已完成。 |
| TASK-7.03 | partial | usage 记录已完成，非流式与流式 final usage source 已区分；后续 manual adjustment 等来源随后台/人工调整能力扩展。 |
| TASK-7.04 | partial | ledger credit/debit、reservation freeze/capture/release、部分余额授权和平台差额核销已完成；外部事务内 debit 幂等重入已完成。 |
| TASK-7.05 | partial | 非流式 settlement、调用上游前 authorization baseline、部分余额授权、平台差额核销和 request-level settlement 成功重放检查已完成；首次 settlement 失败 recovery 未完成。 |
| TASK-7.06 | done | OpenAI stream final usage 解析已完成。 |
| TASK-7.07 | partial | stream 有 final usage 时可 settlement，调用上游前 authorization baseline、部分余额授权、平台差额核销、无 final usage 风险敞口记录、request/attempt 状态机守卫和 request-level settlement 成功重放检查已接入；首次 settlement 失败 recovery 未完成。 |
| TASK-7.17 | done | gateway request-level authorization、capture/release baseline、部分余额授权、平台差额核销、无 final usage 风险敞口记录和 provider/model 输入 token 估算已完成；[GAP-7-004](../../production/TODO_REGISTER.md#gap-7-004)、[GAP-7-013](../../production/TODO_REGISTER.md#gap-7-013)、[GAP-7-014](../../production/TODO_REGISTER.md#gap-7-014) 已关闭。 |
| TASK-7.18 | done | request_records 和 request_attempts 已增加 SQL 原子状态机守卫；重复终态更新会读回第一次终态事实，跨终态覆盖会返回 `requestlog_invalid_state_transition`；[GAP-7-003](../../production/TODO_REGISTER.md#gap-7-003) 已关闭。 |
| TASK-7.21 | done | safe/internal error 和 usage source 审计已完成；[GAP-7-005](../../production/TODO_REGISTER.md#gap-7-005)、[GAP-7-008](../../production/TODO_REGISTER.md#gap-7-008) 已关闭。 |

## 当前进行

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.20 | in_progress | provider/channel 成本价 schema、cost snapshot schema、sqlc 查询和 billing 客户售价/成本价语义拆分已完成；下一步把成本价查询和 `cost_snapshots` 写入接入 settlement。 |

## 未完成

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.19 | partial | 上游成功后的首次 settlement 失败 recovery 暂不插队，后续进入 worker/recovery 线时处理。 |
| TASK-7.20 | partial | provider/channel 成本价和 cost snapshot 表结构已落地；结算层写入请求级成本快照仍未完成。 |
| TASK-7.22 | todo | price effective window 约束。 |

## 下一次进入本阶段前必须检查

```bash
rg -n "GAP-7-" docs/production/TODO_REGISTER.md cmd internal migrations sql
```

## 下节课 TODO

1. 复核当前阶段剩余 P0 release blocker。
2. [GAP-7-009](../../production/TODO_REGISTER.md#gap-7-009) 已开始：成本价/cost snapshot schema、查询和 billing 语义拆分已完成；下一步接入 settlement 写 `cost_snapshots`。
3. [GAP-7-007](../../production/TODO_REGISTER.md#gap-7-007) 仍保留为 worker recovery 阻断项，settlement recovery 暂不做，等进入 worker/settlement recovery 线时处理。
