# Phase 6 Status

状态：partial

## 已完成

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-6.01 | done | provider/channel/model/channel_model schema 和基础查询已完成。 |

## 未完成

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-6.02 | deferred | 正式 credential resolver 推迟到阶段 9 前置。 |
| TASK-6.03 | todo | main 装配仍较重，adapter key 缺少启动前校验。 |
| TASK-6.04 | partial | project 模型 allow-list/deny-list 已接入 routing 和 `/v1/models`；project 禁用、预算和专属 channel 策略未完成。 |
| TASK-6.05 | done | routing 已区分模型不存在、project 不可用和无可用 channel；HTTP handler 已映射为安全 OpenAI-compatible 错误。 |

## 下一次进入本阶段前必须检查

```bash
rg -n "GAP-6-" docs/production/TODO_REGISTER.md cmd internal migrations sql
```
