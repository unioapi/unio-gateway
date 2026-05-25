# Phase 7 Status

状态：in_progress

## 已完成

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.01 | partial | 客户侧 price schema 已建立，成本价和生效窗口约束未完成。 |
| TASK-7.02 | partial | request/attempt 记录已完成，状态机守卫未完成。 |
| TASK-7.03 | partial | usage 记录已完成，usage source 需要细分。 |
| TASK-7.04 | partial | ledger credit/debit、reservation freeze/capture/release、部分余额授权和平台差额核销已完成；幂等未完成。 |
| TASK-7.05 | partial | 非流式 settlement、调用上游前 authorization baseline、部分余额授权和平台差额核销已完成；settlement 请求级幂等未完成。 |
| TASK-7.06 | done | OpenAI stream final usage 解析已完成。 |
| TASK-7.07 | partial | stream 有 final usage 时可 settlement，调用上游前 authorization baseline、部分余额授权、平台差额核销和无 final usage 风险敞口记录已接入；状态机守卫仍未完成。 |
| TASK-7.17 | done | gateway request-level authorization、capture/release baseline、部分余额授权、平台差额核销、无 final usage 风险敞口记录和 provider/model 输入 token 估算已完成；[GAP-7-004](../../production/TODO_REGISTER.md#gap-7-004)、[GAP-7-013](../../production/TODO_REGISTER.md#gap-7-013)、[GAP-7-014](../../production/TODO_REGISTER.md#gap-7-014) 已关闭。 |

## 当前进行

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.18 | todo | 下一步进入 request/attempt 状态机守卫，优先处理 [GAP-7-003](../../production/TODO_REGISTER.md#gap-7-003)。 |

## 未完成

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.18 | todo | request/attempt 状态机守卫。 |
| TASK-7.19 | todo | settlement 幂等；上游成功后的 settlement 失败 recovery 暂不插队，后续进入 worker/recovery 线时处理。 |
| TASK-7.20 | todo | provider/channel 成本价和 cost snapshot。 |
| TASK-7.21 | todo | safe/internal error 与 usage source 审计。 |
| TASK-7.22 | todo | price effective window 约束。 |

## 下一次进入本阶段前必须检查

```bash
rg -n "GAP-7-" docs/production/TODO_REGISTER.md cmd internal migrations sql
```

## 下节课 TODO

1. 进入 request/attempt 状态机守卫前，复核当前阶段所有 P0 release blocker。
2. 优先处理 [GAP-7-003](../../production/TODO_REGISTER.md#gap-7-003)。
3. settlement recovery 暂不做，等进入 worker/settlement 幂等线时处理。
