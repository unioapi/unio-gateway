# Phase 7 Handoff - Billing Ledger

更新时间：2026-05-23

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
14. `billing.Service.EstimateAuthorizationAmount` 已实现，可按预估 prompt/max completion 和当前价格计算调用上游前需要冻结的金额。
15. `ChatAuthorizer` 已装入 `ChatCompletionService`，非流式和流式调用上游前都会创建 request-level authorization。
16. 非流式成功 settlement 会带上同一笔 `ChatAuthorization`；取消、non-retryable error、fallback 全失败、fallback adapter missing 会 release。
17. 流式成功 settlement 会带上同一笔 `ChatAuthorization`；取消、emit 后 error、non-retryable error、missing final usage、fallback 全失败、fallback adapter missing 会 release。
18. gateway authorization 行为测试已覆盖成功、不调用 adapter、fallback、取消、release、settlement failure 等关键路径。

仍需收口：

1. 部分余额授权和平台差额核销，见 [GAP-7-014](../../production/TODO_REGISTER.md#gap-7-014)。
2. provider/model tokenizer，替换 prompt token 临时估算，见 [GAP-7-013](../../production/TODO_REGISTER.md#gap-7-013)。
3. 无 final usage 的商业策略和异常风控，见 [GAP-7-004](../../production/TODO_REGISTER.md#gap-7-004)。
4. request/attempt 状态机守卫。
5. settlement 请求级幂等。
6. error message 和 usage source 审计字段。
7. 成本价和价格生效窗口。

## 已定死的新方案

最终产品规则：**部分余额授权 + 平台差额核销，不允许用户负余额或隐性欠费。**

核心语义：

```text
estimated_amount  = 平台按请求、模型、价格和 max_tokens 估算的风险金额
authorized_amount = 实际从用户可用余额里冻结的金额
actual_amount     = 上游成功后根据真实 usage 算出的应收金额
captured_amount   = min(actual_amount, authorized_amount)
written_off_amount = max(actual_amount - captured_amount, 0)
```

授权阶段：

1. `available_balance <= 0`：拒绝请求，不调用上游。
2. `available_balance >= estimated_amount`：冻结 `estimated_amount`。
3. `0 < available_balance < estimated_amount`：冻结全部 `available_balance`，请求仍可继续。

结算阶段：

1. `actual_amount <= authorized_amount`：capture `actual_amount`，release 多余冻结金额。
2. `actual_amount > authorized_amount`：capture `authorized_amount`，差额写为平台核销；上游成功且有可靠 usage 时 request 仍应成功。
3. 不允许把差额变成用户隐性欠费，不允许用户余额变负，不允许充值后偷偷追扣旧账。

例子：

```text
用户可用余额 0.80
预估需要 1.00
实际花费 1.00

授权：冻结 0.80，请求继续
结算：扣用户 0.80，平台核销 0.20
用户最终看到余额 0
request succeeded
```

当前代码还不是这个逻辑：

1. `ChatAuthorizationService.AuthorizeChat` 仍要求按 estimated amount 全额冻结，余额不足会直接失败。
2. `ledger.captureWithQueries` 仍拒绝 `actual_amount > authorized_amount`。
3. 缺少 write-off / platform loss 的账务事实表或字段。

## 下一步

下一节第一步：

```text
7.17 余额预检查与冻结闭环：实现 GAP-7-014 部分余额授权和平台差额核销。
```

建议接入顺序：

1. 设计 write-off 账务事实：至少能记录 request_record_id、reservation_id、actual_amount、captured_amount、written_off_amount、currency、reason。
2. authorization 拆分 `estimated_amount` 与 `authorized_amount`，低余额时冻结可用余额而不是直接失败。
3. settlement 拆分 actual amount 和 capture amount；actual 超过 authorized 时 capture 冻结金额并写 write-off。
4. request 成功收口必须和 usage、price snapshot、capture、write-off 同事务提交。
5. 补非流式和流式测试：低余额放行、余额为 0 拒绝、actual 小于/等于/大于冻结金额。

必须先处理的 GAP：

- [GAP-7-014](../../production/TODO_REGISTER.md#gap-7-014)
- [GAP-7-013](../../production/TODO_REGISTER.md#gap-7-013)
- [GAP-7-004](../../production/TODO_REGISTER.md#gap-7-004)

相关文档：

1. [PLAN.md](PLAN.md#task-7-17-preauthorization)
2. [STATUS.md](STATUS.md)
3. [ACCEPTANCE.md](ACCEPTANCE.md)
4. [TODO_REGISTER.md](../../production/TODO_REGISTER.md)
5. [RELEASE_BLOCKERS.md](../../production/RELEASE_BLOCKERS.md)
6. [DECISIONS.md](../../production/DECISIONS.md#dec-006-部分余额放行与平台差额核销)

当前关键文件：

1. `internal/billing/service.go`
2. `internal/billing/types.go`
3. `internal/ledger/reservation.go`
4. `internal/gateway/chat_authorization.go`
5. `internal/gateway/chat_completion.go`
6. `internal/gateway/chat_stream.go`
7. `internal/gateway/chat_settlement.go`
8. `internal/gateway/service_test.go`
9. `sql/queries/ledger.sql`
10. `migrations/000007_create_ledger_tables.up.sql`
11. `migrations/000009_create_ledger_reservations.up.sql`

## 注意事项

1. 不要退回“用户必须自己算 token 才能调用”的产品体验。
2. 不要实现隐性欠费、负余额或充值后追扣；如果未来要做信用额度，必须另开决策和账务模型。
3. `estimated_amount` 和 `authorized_amount` 是两个概念；当前代码把它们等同，是 [GAP-7-014](../../production/TODO_REGISTER.md#gap-7-014) 要修的核心。
4. 上游成功且有可靠 usage 时，`actual_amount > authorized_amount` 不应再导致普通 settlement failed。
5. write-off 必须是可审计账务事实，不能只写日志。
6. stream 不能只复用非流式后扣费模型，需要 authorization、release、capture 和 write-off 语义。
7. ledger-first 不能被绕过，余额变化和核销都必须有账务事实。
8. reservation 表方案已经落地，下一步不要再重开“reservation 表还是 reservation ledger”的设计。
9. 所有补偿和重试都要考虑幂等。
10. `Capture` 的 0 金额场景应走 `Release`，不是写 0 金额 ledger entry。

## 最近验证

最近一次全量验证：

```bash
go test -count=1 ./...
```

结果：通过，时间为 2026-05-23。
