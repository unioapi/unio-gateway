# Phase 7 Status

状态：in_progress

## 已完成

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.01 | done | 客户侧 price schema、provider/channel 成本价、请求级 cost snapshot 和价格生效窗口约束已完成；[GAP-7-009](../../production/TODO_REGISTER.md#gap-7-009)、[GAP-7-010](../../production/TODO_REGISTER.md#gap-7-010) 已关闭。 |
| TASK-7.02 | done | request/attempt 记录、状态机守卫、safe error message 和 internal error detail 审计字段已完成。 |
| TASK-7.03 | partial | usage 记录已完成，非流式与流式 final usage source 已区分；后续 manual adjustment 等来源随后台/人工调整能力扩展。 |
| TASK-7.04 | partial | ledger credit/debit、reservation freeze/capture/release、部分余额授权和平台差额核销已完成；外部事务内 debit 幂等重入已完成。 |
| TASK-7.05 | partial | 非流式 settlement、调用上游前 authorization baseline、部分余额授权、平台差额核销和 request-level settlement 成功重放检查已完成；首次 settlement 失败 recovery 未完成。 |
| TASK-7.06 | done | OpenAI stream final usage 解析已完成。 |
| TASK-7.07 | partial | stream 有 final usage 时可 settlement，调用上游前 authorization baseline、部分余额授权、平台差额核销、无 final usage 风险敞口记录、request/attempt 状态机守卫和 request-level settlement 成功重放检查已接入；首次 settlement 失败 recovery 未完成。 |
| TASK-7.17 | done | gateway request-level authorization、capture/release baseline、部分余额授权、平台差额核销、无 final usage 风险敞口记录和 provider/model 输入 token 估算已完成；[GAP-7-004](../../production/TODO_REGISTER.md#gap-7-004)、[GAP-7-013](../../production/TODO_REGISTER.md#gap-7-013)、[GAP-7-014](../../production/TODO_REGISTER.md#gap-7-014) 已关闭。 |
| TASK-7.18 | done | request_records 和 request_attempts 已增加 SQL 原子状态机守卫；重复终态更新会读回第一次终态事实，跨终态覆盖会返回 `requestlog_invalid_state_transition`；[GAP-7-003](../../production/TODO_REGISTER.md#gap-7-003) 已关闭。 |
| TASK-7.20 | done | provider/channel 成本价 schema、cost snapshot schema、sqlc 查询、billing 客户售价/成本价语义拆分、settlement 写入请求级 `cost_snapshots` 和幂等重放校验已完成；[GAP-7-009](../../production/TODO_REGISTER.md#gap-7-009) 已关闭。 |
| TASK-7.21 | done | safe/internal error 和 usage source 审计已完成；[GAP-7-005](../../production/TODO_REGISTER.md#gap-7-005)、[GAP-7-008](../../production/TODO_REGISTER.md#gap-7-008) 已关闭。 |
| TASK-7.22 | done | prices 已通过 PostgreSQL exclusion constraint 防止同一 model/currency/pricing_unit 出现重叠 enabled 生效窗口；[GAP-7-010](../../production/TODO_REGISTER.md#gap-7-010) 已关闭。 |

## 当前进行

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| 无 | - | 当前建议下一步进入 TASK-7.19 worker/settlement recovery。 |

## 未完成

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.19 | partial | 上游成功后的首次 settlement 失败 recovery 暂不插队，后续进入 worker/recovery 线时处理。 |

## 下一次进入本阶段前必须检查

```bash
rg -n "GAP-7-" docs/production/TODO_REGISTER.md cmd internal migrations sql
```

## 下节课 TODO

1. 复核当前阶段剩余 P0 release blocker。
2. 继续推进 [GAP-7-007](../../production/TODO_REGISTER.md#gap-7-007) worker recovery。
3. 后续处理 [GAP-7-006](../../production/TODO_REGISTER.md#gap-7-006) stream 写出后错误观测。
