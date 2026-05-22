# Phase 7 Handoff - Billing Ledger

更新时间：2026-05-22

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
11. ledger reservation 表已创建，并已接入 `user_balances.reserved_balance`。
12. `ledger.Service` 已拆出 `entry.go`、`reservation.go`、`numeric.go`、`convert.go`、`errors.go`、`constant.go`。
13. `ledger.Service.PreAuthorize`、`Capture`、`Release` 已实现，支持冻结、真实扣费、释放和幂等重入。
14. `billing.Service.EstimateAuthorization` 已实现，可按预估 prompt/max completion 和当前价格计算预授权冻结金额。

仍需收口：

1. gateway 调用上游前的余额预检查与预授权接入。
2. stream freeze/capture/release。
3. request/attempt 状态机守卫。
4. settlement 请求级幂等。
5. 无 final usage 的商业策略。
6. error message 和 usage source 审计字段。
7. 成本价和价格生效窗口。

## 下一步

下一节第一步：

```text
先拆分 internal/billing/service.go，保持行为不变。
```

拆分建议：

1. 保留 `billing.Service` 对外方法签名不变。
2. 把 DTO/常量、真实 usage 结算、预授权估算、价格规范化、NUMERIC helper 分到独立文件。
3. 拆分后先跑 `go test ./internal/billing` 和 `go test ./...`。
4. 这是教学/重构 TODO，不是 production GAP，不需要登记到 TODO_REGISTER。

拆分完成后继续：

```text
7.17 余额预检查与预授权最小闭环：接入 gateway preauthorization。
```

必须先处理的 GAP：

- [GAP-7-001](../../production/TODO_REGISTER.md#gap-7-001)
- [GAP-7-002](../../production/TODO_REGISTER.md#gap-7-002)
- [GAP-7-004](../../production/TODO_REGISTER.md#gap-7-004)
- [GAP-7-011](../../production/TODO_REGISTER.md#gap-7-011)


相关文档：

1. [PLAN.md](PLAN.md#task-7-17-preauthorization)
2. [STATUS.md](STATUS.md)
3. [ACCEPTANCE.md](ACCEPTANCE.md)
4. [TODO_REGISTER.md](../../production/TODO_REGISTER.md)
5. [RELEASE_BLOCKERS.md](../../production/RELEASE_BLOCKERS.md)

当前关键文件：

1. `internal/billing/service.go`
2. `internal/ledger/reservation.go`
3. `internal/gateway/chat_completion.go`
4. `internal/gateway/chat_stream.go`
5. `internal/gateway/chat_settlement.go`
6. `sql/queries/ledger.sql`
7. `migrations/000007_create_ledger_tables.up.sql`
8. `migrations/000009_create_ledger_reservations.up.sql`

## 注意事项

1. 不要只做“余额不足时 settlement 失败”，这不能阻止平台先产生上游成本。
2. stream 不能只复用非流式后扣费模型，需要预授权和 release 语义。
3. ledger-first 不能被绕过，余额变化必须有账本事实。
4. reservation 表方案已经落地，下一步不要再重开“reservation 表还是 reservation ledger”的设计。
5. 所有补偿和重试都要考虑幂等。
6. `Capture` 的 0 金额场景应走 `Release`，不是写 0 金额 ledger entry。

## 最近验证

最近一次全量验证：

```bash
go test ./...
```

结果：通过，时间为 2026-05-22。
