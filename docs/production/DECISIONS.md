# Decisions

本文档记录会影响后续实现和商业语义的关键决策。

## DEC-001 个人账户余额先落 user

状态：accepted

决策：

```text
当前个人账户模式下，余额事实落在 user_balances 和 ledger_entries.user_id。
project 只作为应用空间、API key 容器、用量归集和未来预算边界。
```

原因：

```text
现阶段没有 organization/billing_account。把余额先落 user 可以避免过早引入团队账务模型。
```

影响：

```text
request_records 必须同时记录 user_id、project_id、api_key_id，保证扣费、审计和统计都可追溯。
```

## DEC-002 Adapter 不读取 provider/channel 配置

状态：accepted

决策：

```text
adapter 只接收 channel.Runtime，不读取 env，不查询数据库，不保存业务状态。
```

原因：

```text
provider、channel、model、price、credential 属于业务数据，后续必须由后台管理和数据库驱动。
```

影响：

```text
gateway/routing 负责选择 channel 并解析 credential，adapter 只负责协议转换和上游 HTTP 调用。
```

## DEC-003 Stream 无 final usage 暂不扣费

状态：accepted

决策：

```text
第 7 阶段在没有 final usage 的 stream 请求中不强行按已输出 chunk 估算扣费。
```

原因：

```text
估算扣费可能导致误扣，且当前没有余额冻结和 release 闭环。
```

影响：

```text
公开生产前必须实现余额冻结、异常状态记录和风控策略。
关联任务：../chapters/phase-07-billing-ledger/PLAN.md#task-7-17-preauthorization
```

## DEC-004 request_id 与 correlation id 分离

状态：accepted

决策：

```text
request_records.request_id 使用服务端生成的业务请求 ID。
HTTP X-Request-ID 只作为 correlation id 用于日志/链路关联。
```

原因：

```text
客户端可控 ID 不适合作为账务和请求记录事实主键。
```

影响：

```text
业务请求记录必须由 requestlog.GenerateRequestID 创建。
HTTP correlation id 仍需要输入约束。
```

## DEC-005 第三方库选择不以“少用”为目标

状态：accepted

决策：

```text
业务核心逻辑保持自研可审计；通用基础设施、安全、协议解析、可观测性和精度计算优先评估成熟库。
```

原因：

```text
商业项目的目标不是展示手写能力，而是在核心边界可控的前提下降低维护风险。
```

影响：

```text
新增通用能力前必须先检查 docs/production/THIRD_PARTY_POLICY.md。
```

## DEC-006 部分余额放行与平台差额核销

状态：accepted

决策：

```text
Chat 请求采用严格不透支用户余额的预付费模型，但低余额用户可以消耗剩余可用余额。

调用上游前，billing 计算 estimated_amount；ledger 实际冻结 authorized_amount。
当 available_balance >= estimated_amount 时，authorized_amount = estimated_amount。
当 0 < available_balance < estimated_amount 时，authorized_amount = available_balance，请求仍可继续。
当 available_balance <= 0 时，请求直接拒绝，不调用上游。

请求成功后按真实 actual_amount 结算：
captured_amount = min(actual_amount, authorized_amount)
written_off_amount = max(actual_amount - captured_amount, 0)

written_off_amount 是平台差额核销，不形成用户隐性欠费，不允许用户余额变负。
```

原因：

```text
用户不应该为了发起 API 请求而精确计算 token 或预估费用。
如果要求余额必须覆盖最坏冻结金额，低余额用户会出现“有余额但花不出去”的反人类体验。
如果允许追扣或负余额，又会引入欠费、追款和充值抵扣等新的产品与账务复杂度。

当前阶段选择平台承担少量估算差额，用 tokenizer、模型 max_tokens、核销上限和告警把风险压到低频异常。
```

影响：

```text
authorization 必须拆分 estimated_amount 与 authorized_amount。
settlement 不能再把 actual_amount > authorized_amount 当作普通失败；
应 capture 已冻结金额，记录 write-off 账务事实，并在上游成功且有 usage 时让 request 成功收口。

该规则已由 [GAP-7-014](TODO_REGISTER.md#gap-7-014) 在 2026-05-25 落地，后续仍需用 tokenizer、模型 max_tokens、核销上限和告警继续降低平台风险。
```
