# Phase 7 Handoff - Billing Ledger

更新时间：2026-05-19

## 当前状态

阶段 7 尚未完成，不应进入阶段 8。

已经完成：

1. request record 和 attempt record 基础链路。
2. usage record 基础链路。
3. price snapshot 基础链路。
4. ledger credit/debit 基础链路。
5. 非流式 settlement。
6. OpenAI stream final usage 解析。
7. adapter stream contract 扩展 usage chunk。
8. stream 有 final usage 时的 settlement。
9. cached token 和 reasoning token 进入 usage 和 billing。
10. request_id 与 correlation id 分离。

仍需收口：

1. 余额预检查与预授权。
2. stream freeze/capture/refund。
3. request/attempt 状态机守卫。
4. settlement 请求级幂等。
5. 无 final usage 的商业策略。
6. error message 和 usage source 审计字段。
7. 成本价和价格生效窗口。

## 下一步

下一节建议：

```text
7.17 余额预检查与预授权最小闭环
```

必须先处理的 GAP：

```text
GAP-7-001
GAP-7-002
GAP-7-004
GAP-7-011
```

相关文档：

1. [PLAN.md](PLAN.md#task-7-17-preauthorization)
2. [STATUS.md](STATUS.md)
3. [ACCEPTANCE.md](ACCEPTANCE.md)
4. [TODO_REGISTER.md](../../production/TODO_REGISTER.md)
5. [RELEASE_BLOCKERS.md](../../production/RELEASE_BLOCKERS.md)

## 注意事项

1. 不要只做“余额不足时 settlement 失败”，这不能阻止平台先产生上游成本。
2. stream 不能只复用非流式后扣费模型，需要预授权和 refund 语义。
3. ledger-first 不能被绕过，余额变化必须有账本事实。
4. preauthorization 设计前要先决定 reservation 表还是 reservation ledger。
5. 所有补偿和重试都要考虑幂等。

