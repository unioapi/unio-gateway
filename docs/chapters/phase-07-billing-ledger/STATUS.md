# Phase 7 Status

状态：in_progress

## 已完成

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.01 | partial | 客户侧 price schema 已建立，成本价和生效窗口约束未完成。 |
| TASK-7.02 | partial | request/attempt 记录已完成，状态机守卫未完成。 |
| TASK-7.03 | partial | usage 记录已完成，usage source 需要细分。 |
| TASK-7.04 | partial | ledger credit/debit 已完成，preauthorize/capture/refund 未完成。 |
| TASK-7.05 | partial | 非流式 settlement 已完成，余额预授权和幂等未完成。 |
| TASK-7.06 | done | OpenAI stream final usage 解析已完成。 |
| TASK-7.07 | partial | stream 有 final usage 时可 settlement，无 final usage 策略仍需生产化。 |

## 当前进行

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-7.17 | todo | 下一步应先设计并落地余额预检查与预授权最小闭环。 |

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

