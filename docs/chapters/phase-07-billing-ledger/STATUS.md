# Phase 7 Status

状态：in_progress

## 已完成

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.01 | partial | 客户侧 price schema 已建立，成本价和生效窗口约束未完成。 |
| TASK-7.02 | partial | request/attempt 记录已完成，状态机守卫未完成。 |
| TASK-7.03 | partial | usage 记录已完成，usage source 需要细分。 |
| TASK-7.04 | partial | ledger credit/debit、reservation freeze/capture/release 已完成，gateway authorization baseline 已接入；部分余额授权、差额核销和幂等未完成。 |
| TASK-7.05 | partial | 非流式 settlement 和调用上游前 authorization baseline 已完成；部分余额授权、差额核销和幂等未完成。 |
| TASK-7.06 | done | OpenAI stream final usage 解析已完成。 |
| TASK-7.07 | partial | stream 有 final usage 时可 settlement，调用上游前 authorization baseline 已接入；无 final usage 策略和差额核销仍需生产化。 |

## 当前进行

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.17 | in_progress | gateway request-level authorization、capture/release baseline 已接入；最终产品规则已定为部分余额授权 + 平台差额核销，尚未落地，关联 [GAP-7-014](../../production/TODO_REGISTER.md#gap-7-014)；prompt token 仍为临时估算，关联 [GAP-7-013](../../production/TODO_REGISTER.md#gap-7-013)。 |

## 未完成

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.18 | todo | request/attempt 状态机守卫。 |
| TASK-7.19 | todo | settlement 幂等。 |
| TASK-7.20 | todo | provider/channel 成本价和 cost snapshot。 |
| TASK-7.21 | todo | safe/internal error 与 usage source 审计。 |
| TASK-7.22 | todo | price effective window 约束。 |

## 下一次进入本阶段前必须检查

```bash
rg -n "GAP-7-" docs/production/TODO_REGISTER.md cmd internal migrations sql
```

## 下节课 TODO

1. 实现部分余额授权：拆分 `estimated_amount` 与 `authorized_amount`，`available_balance > 0` 时冻结可用余额并允许请求继续。
2. 实现平台差额核销：`actual_amount > authorized_amount` 时 capture 已冻结金额，差额写入可审计 write-off 事实，请求按成功收口。
3. 接入 provider/model tokenizer，替换 prompt token 临时估算。
