# Phase 4 Status

状态：partial

## 已完成

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-4.01 | partial | `/v1/models` 已接入 model catalog，但 project 可见性未完成。 |
| TASK-4.02 | done | chat completions HTTP 入口和 stream 分支已完成。 |
| TASK-4.03 | done | Chat DTO 已完成 text-only MVP 深度校验，非法 role/content/参数边界会返回 OpenAI-compatible error。 |
| TASK-4.04 | done | JSON decode 已校验 Content-Type、空 body、超大 body、malformed JSON 和 trailing token，并返回稳定 OpenAI-compatible error。 |
| TASK-4.05 | partial | SSE 写出行为已存在，但写出后错误观测仍需完善。 |

## 仍需处理

| 任务 | 状态 | 说明 |
| --- | --- | --- |
| TASK-4.01 | deferred | `/v1/models` project 可见性依赖阶段 6 project policy，关联 [GAP-6-006](../../production/TODO_REGISTER.md#gap-6-006)。 |
| TASK-4.05 | deferred | SSE 写出后错误观测依赖阶段 7 request 状态和阶段 8 observability，关联 [GAP-7-006](../../production/TODO_REGISTER.md#gap-7-006)。 |

## 下一次进入本阶段前必须检查

```bash
rg -n "GAP-4-" docs/production/TODO_REGISTER.md cmd internal migrations sql
```
