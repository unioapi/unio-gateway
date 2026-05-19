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
| TASK-6.04 | todo | project 级模型可见性和 routing policy 未完成。 |
| TASK-6.05 | todo | routing 错误语义还不够细。 |

## 下一次进入本阶段前必须检查

```bash
rg -n "GAP-6-" docs/production/TODO_REGISTER.md cmd internal migrations sql
```

