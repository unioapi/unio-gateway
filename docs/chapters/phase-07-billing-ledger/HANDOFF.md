# Phase 7 Handoff - Billing Ledger

更新时间：2026-05-25

## 当前状态

阶段 7 尚未完成，不应进入阶段 8。

## 本班次交接重点

本班次围绕计费边界和账务收口做了 6 个生产修正：

1. 非流式 OpenAI usage 缺失不再被当成 0 usage 成功请求。
   - `internal/adapter/openai/dto.go` 将 usage 改为指针语义。
   - 缺少 usage 或 token 字段时，adapter 返回稳定 failure。
2. authorization 和 settlement 不再读取两次不同价格。
   - `ChatAuthorization` 保存 authorization 时的 `PriceID` 和 `billing.PriceSnapshot`。
   - settlement 使用 authorization 快照创建 `price_snapshots` 并计算费用。
3. 余额不足错误已映射成用户友好的 OpenAI-compatible `insufficient_quota`。
   - HTTP 状态码为 429。
   - 内部仍通过 `failure.CodeLedgerInsufficientBalance` 判断。
4. `GAP-7-014` 已关闭。
   - authorization 拆分 `estimated_amount` 与 `authorized_amount`。
   - `available_balance <= 0` 时拒绝。
   - `0 < available_balance < estimated_amount` 时冻结全部可用余额并继续请求。
   - `actual_amount > authorized_amount` 时只 capture 已冻结金额，差额写 `write_off`。
5. `GAP-7-004` 已关闭。
   - 已经可能产生上游成本但没有 final usage 的 stream 路径不扣用户钱。
   - 释放冻结余额，并写 `risk_exposure` 审计事实。
6. 账务异常统一进 `ledger_billing_exceptions`。
   - `event_type = write_off`：真实费用已知但超过冻结金额。
   - `event_type = risk_exposure`：真实费用未知但平台可能已有成本风险。

本班次不要再恢复旧表名 `ledger_write_offs`；后台、报表和后续查询都应该围绕 `ledger_billing_exceptions` 读取。

## 本次新增决策

settlement 成功语义后的失败暂时不在当前小节实现补偿 worker。

具体判断：

```text
上游已经成功并返回可靠 usage 后，如果 SettleSuccessfulChat 失败，不能简单 release 冻结余额。
因为 provider 侧可能已经产生成本，业务语义应优先补偿重试 settlement/capture。

当前接受这个风险暂时存在：
request 会被标记 failed，reservation 可能保持 authorized，reserved_balance 可能悬挂。

该问题不使用 gateway goroutine 处理。
后续进入 worker/settlement recovery 线时，用数据库持久化 recovery job + 幂等 settlement 重试收口。
```

`GAP-7-013` 已关闭。下一节不切到 recovery worker，进入 `GAP-7-003` request/attempt 状态机守卫。

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
17. 流式成功 settlement 会带上同一笔 `ChatAuthorization`；普通失败和 fallback 失败路径会 release，可能产生上游成本但没有 final usage 的路径会 exception release。
18. gateway authorization 行为测试已覆盖成功、不调用 adapter、fallback、取消、release、settlement failure 等关键路径。
19. `GAP-7-014` 已关闭：authorization 已拆分 `estimated_amount` 与 `authorized_amount`，低余额可冻结全部可用余额并继续请求，`actual_amount > authorized_amount` 时 capture 已冻结金额并写入 `ledger_billing_exceptions` 的 `write_off` 平台核销事实。
20. `GAP-7-004` 已关闭：可能产生上游成本但没有 final usage 的 stream 路径会释放用户冻结余额，并写入 `ledger_billing_exceptions` 的 `risk_exposure` 事实。
21. `GAP-7-013` 已关闭：gateway authorization 已通过 adapter registry 调用 provider adapter 注册的 `ChatInputTokenizer`；OpenAI adapter 已用 `tiktoken-go/tokenizer` 按 upstream model 估算 chat 输入 token，旧的字符串长度临时估算已移除。

仍需收口：

1. request/attempt 状态机守卫。
2. settlement 请求级幂等。
3. error message 和 usage source 审计字段。
4. 成本价和价格生效窗口。

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

当前代码已落地这个逻辑：

1. `ChatAuthorizationService.AuthorizeChat` 传入 `estimated_amount`，`ledger.PreAuthorize` 根据可用余额写入真实 `authorized_amount`。
2. `ledger.captureWithQueries` 使用 `captured_amount = min(actual_amount, authorized_amount)`。
3. `actual_amount > authorized_amount` 时会写入 `ledger_billing_exceptions` 的 `write_off` 事件，记录平台核销事实。
4. 无 final usage 的 stream 客户端取消、emit 后中断和正常结束缺 final usage 会写入 `ledger_billing_exceptions` 的 `risk_exposure` 事件。

## 下一步

下一节第一步：

```text
7.18 Request/attempt 状态机守卫：处理 GAP-7-003。
```

建议接入顺序：

1. 进入 request/attempt 状态机守卫和 settlement 幂等前，复核剩余 P0 blocker。
2. 处理 request/attempt 终态更新状态机守卫。
3. 后续后台/报表查询需要同时读取 `ledger_billing_exceptions` 中的 `write_off` 与 `risk_exposure` 事件。
4. settlement recovery 暂时不做，等进入 `cmd/worker` / recovery job 小节时再处理。

必须先处理的 GAP：

- [GAP-7-003](../../production/TODO_REGISTER.md#gap-7-003)

相关文档：

1. [PLAN.md](PLAN.md#task-7-17-preauthorization)
2. [STATUS.md](STATUS.md)
3. [ACCEPTANCE.md](ACCEPTANCE.md)
4. [TODO_REGISTER.md](../../production/TODO_REGISTER.md)
5. [RELEASE_BLOCKERS.md](../../production/RELEASE_BLOCKERS.md)
6. [DECISIONS.md](../../production/DECISIONS.md#dec-006-部分余额放行与平台差额核销)
7. [DECISIONS.md](../../production/DECISIONS.md#dec-007-settlement-失败补偿归属-worker)

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
3. `estimated_amount` 和 `authorized_amount` 是两个概念；当前代码已拆分二者，后续不要重新把预估金额等同于实际冻结金额。
4. 上游成功且有可靠 usage 时，`actual_amount > authorized_amount` 不应再导致普通 settlement failed。
5. write-off 必须是可审计账务事实，不能只写日志。
6. stream 不能只复用非流式后扣费模型，需要 authorization、release、capture、write-off 和 risk-exposure 语义。
7. ledger-first 不能被绕过，余额变化和核销都必须有账务事实。
8. reservation 表方案已经落地，下一步不要再重开“reservation 表还是 reservation ledger”的设计。
9. 所有补偿和重试都要考虑幂等。
10. `Capture` 的 0 金额场景应走 `Release`，不是写 0 金额 ledger entry。
11. 上游成功且有可靠 usage 后 settlement 失败，不要直接 release；后续必须用 worker 持久化补偿任务和幂等 settlement 重试处理。

## 最近验证

最近一次全量验证：

```bash
go test -count=1 ./...
```

结果：通过，时间为 2026-05-25。
