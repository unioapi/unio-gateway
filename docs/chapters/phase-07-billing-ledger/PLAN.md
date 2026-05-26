# Phase 7 Plan - 计费与账本

## 目标

建立商业 API 网关的计费事实链路。

本阶段必须让每次请求能追溯：

```text
用户身份
项目和 API key
请求记录
上游 attempt
模型/provider/channel
usage
price snapshot
cost snapshot
ledger entry
余额变化
```

## 已完成主线

<a id="task-7-01-price-schema"></a>
### TASK-7.01 Price schema

状态：partial

范围：

1. 建立客户侧价格表。
2. 支持 input/output/cached/reasoning token pricing unit。
3. 后续补 provider/channel 成本价和 request-level cost snapshot。

关联 GAP：

- [GAP-7-009](../../production/TODO_REGISTER.md#gap-7-009)
- [GAP-7-010](../../production/TODO_REGISTER.md#gap-7-010)


<a id="task-7-02-request-attempt-record"></a>
### TASK-7.02 Request record 与 attempt record

状态：partial

范围：

1. 创建 request record。
2. 创建 attempt record。
3. 记录 user/project/api_key/model/provider/channel。
4. 记录 succeeded/failed/canceled 状态。
5. request/attempt 状态机守卫已完成，终态不会被并发补偿或重复更新覆盖。

关联 GAP：

- [GAP-7-003](../../production/TODO_REGISTER.md#gap-7-003) 已关闭
- [GAP-7-005](../../production/TODO_REGISTER.md#gap-7-005) 已关闭


<a id="task-7-03-usage-record"></a>
### TASK-7.03 Usage record

状态：partial

范围：

1. 记录 prompt/completion/total token。
2. 记录 cached/reasoning token。
3. 后续区分非流式 response 与 stream final usage 来源。

关联 GAP：

- [GAP-7-008](../../production/TODO_REGISTER.md#gap-7-008) 已关闭


<a id="task-7-04-ledger-debit"></a>
### TASK-7.04 Ledger debit

状态：partial

范围：

1. 支持 credit/debit ledger entry。
2. debit 使用事务更新 user balance。
3. ledger reservation pre-authorize/capture/release 已完成，并已接入 gateway request-level authorization。
4. 后续补部分余额授权、差额核销和 settlement 幂等。

关联 GAP：

- [GAP-7-011](../../production/TODO_REGISTER.md#gap-7-011)
- [GAP-7-012](../../production/TODO_REGISTER.md#gap-7-012)


<a id="task-7-05-non-stream-settlement"></a>
### TASK-7.05 非流式 settlement

状态：partial

范围：

1. adapter 返回 usage 后创建 usage record。
2. 创建 price snapshot。
3. 创建 ledger debit。
4. 标记 request/attempt succeeded。
5. 调用上游前 request-level authorization 已接入。
6. 后续补部分余额授权、差额核销和 settlement 幂等。

关联 GAP：

- [GAP-7-001](../../production/TODO_REGISTER.md#gap-7-001)
- [GAP-7-007](../../production/TODO_REGISTER.md#gap-7-007)
- [GAP-7-012](../../production/TODO_REGISTER.md#gap-7-012)


<a id="task-7-06-stream-final-usage"></a>
### TASK-7.06 OpenAI stream final usage

状态：done

范围：

1. OpenAI stream request 设置 `stream_options.include_usage=true`。
2. adapter 解析 final usage chunk。
3. adapter stream contract 承载 usage-only chunk。
4. gateway 捕获 final usage，不向客户端转发 usage-only chunk。

<a id="task-7-07-stream-settlement"></a>
### TASK-7.07 Stream settlement

状态：partial

范围：

1. 有 final usage 时执行 settlement。
2. 客户端取消但已拿到 final usage 时仍 settlement。
3. 无 final usage 时不强行估算扣费；已经可能产生上游成本的路径释放用户冻结余额，并记录 `risk_exposure`。
4. 调用上游前 request-level authorization 已接入。
5. request/attempt 状态机守卫已完成；后续补 settlement 幂等和观测字段。

关联 GAP：

- [GAP-7-002](../../production/TODO_REGISTER.md#gap-7-002)
- [GAP-7-006](../../production/TODO_REGISTER.md#gap-7-006)
- [GAP-7-011](../../production/TODO_REGISTER.md#gap-7-011)


## 当前必须收口任务

<a id="task-7-17-preauthorization"></a>
### TASK-7.17 余额预检查与冻结闭环

状态：done

范围：

1. 请求调用上游前检查用户可用余额。
2. 非流式和流式请求已接入 request-level authorization。
3. 成功后按真实 usage capture，失败、取消、无 final usage 时 release 或进入异常状态。
4. 最终产品规则必须拆分 `estimated_amount` 与 `authorized_amount`。
5. 当 `available_balance <= 0` 时，调用上游前拒绝。
6. 当 `0 < available_balance < estimated_amount` 时，冻结全部可用余额并允许请求继续。
7. 当 `actual_amount > authorized_amount` 时，capture 已冻结金额，差额记为平台 `written_off_amount` / `platform_loss`，上游成功且有 usage 的请求仍应成功收口。
8. 为后续 project 预算或用量上限预留统一判断入口。
9. 所有余额、核销和异常动作都必须可审计。

已完成：

```text
authorization 已拆分 estimated_amount 与 authorized_amount。
available_balance <= 0 时仍会在调用上游前拒绝。
0 < available_balance < estimated_amount 时会冻结全部可用余额并允许请求继续。
actual_amount > authorized_amount 时会 capture 已冻结金额，并写入 ledger_billing_exceptions 的 write_off 事件作为平台核销事实。
stream 已经可能产生上游成本但没有 final usage 时会 release 用户冻结余额，并写入 ledger_billing_exceptions 的 risk_exposure 事件。
上游成功且有可靠 usage 的请求可按成功账务事实收口，不形成用户负余额或隐性欠费。
gateway authorization 已通过 adapter registry 调用 provider adapter 注册的 ChatInputTokenizer。
OpenAI adapter 已用 tiktoken-go/tokenizer 按 upstream model 估算 chat 输入 token。
```

剩余风险：

```text
无。后续新增 provider adapter 时必须注册自己的 ChatInputTokenizer，否则 authorization 会拒绝请求。
```

关联 GAP：

- [GAP-7-013](../../production/TODO_REGISTER.md#gap-7-013)

已关闭 GAP：

- [GAP-7-001](../../production/TODO_REGISTER.md#gap-7-001)
- [GAP-7-002](../../production/TODO_REGISTER.md#gap-7-002)
- [GAP-7-004](../../production/TODO_REGISTER.md#gap-7-004)
- [GAP-7-011](../../production/TODO_REGISTER.md#gap-7-011)
- [GAP-7-013](../../production/TODO_REGISTER.md#gap-7-013)
- [GAP-7-014](../../production/TODO_REGISTER.md#gap-7-014)


<a id="task-7-18-request-state-machine"></a>
### TASK-7.18 Request/attempt 状态机守卫

状态：done

范围：

1. 明确 pending、succeeded、failed、canceled 等状态转移。
2. SQL 更新必须带合法前置状态条件。
3. 终态不能被补偿任务或并发重试覆盖。
4. 重复终态更新应具备幂等语义。

已完成：

```text
request_records 已用 SQL 原子状态机守卫 pending -> running、running -> succeeded/failed/canceled。
request_attempts 已用 SQL 原子状态机守卫 running -> succeeded/failed/canceled。
重复写入同一终态会读回第一次终态事实，不覆盖已有 response/error/upstream metadata。
跨终态覆盖会返回 no rows，并由 requestlog.Store 映射为 requestlog_invalid_state_transition。
sqlc 测试已覆盖 request/attempt 终态幂等和非法覆盖场景。
```

关联 GAP：

- [GAP-7-003](../../production/TODO_REGISTER.md#gap-7-003)


<a id="task-7-19-settlement-idempotency"></a>
### TASK-7.19 Settlement 幂等

状态：partial

范围：

1. 以 request_record_id 作为 settlement 幂等边界。已完成。
2. 重复 settlement 时检测既有 usage/snapshot/ledger。已完成。
3. 并发 settlement 不能把已成功请求误标失败。已完成。
4. 外部事务内的 ledger 幂等冲突需要稳定重入策略。已完成。
5. 上游成功且有可靠 usage 后，settlement 失败不能直接 release 冻结余额。仍保留为 worker recovery 问题。
6. settlement recovery 后续应由 worker 持久化任务处理，不使用 gateway goroutine。未完成。
7. recovery worker 重试 settlement 前，必须先保证 usage、price snapshot、ledger capture 和 request 状态更新幂等。成功重放和外部 debit 幂等已完成，持久化 recovery job 未完成。

关联 GAP：

- [GAP-7-007](../../production/TODO_REGISTER.md#gap-7-007)
- [GAP-7-012](../../production/TODO_REGISTER.md#gap-7-012) 已关闭


<a id="task-7-20-cost-snapshot"></a>
### TASK-7.20 成本价与毛利审计

状态：todo

定价原则：

```text
倍率不是账务事实，只能作为后续后台运营配置工具。
本阶段先落地明确金额的成本价和请求级 cost snapshot。
结算层只消费明确金额，不在 settlement 时依赖模型倍率、补全倍率或分组倍率临时重算历史账单。
```

范围：

1. 区分客户侧售卖价和 provider/channel 成本价。
2. request settlement 保存 cost snapshot。
3. 支持按 channel/provider 分析毛利和 fallback 成本。
4. 后续后台可以支持倍率生成价格，但必须先解析成明确金额后再进入结算。
5. 历史账单复算以 price snapshot 和 cost snapshot 为准，不以当前倍率配置为准。

关联 GAP：

- [GAP-7-009](../../production/TODO_REGISTER.md#gap-7-009)


<a id="task-7-21-error-usage-audit"></a>
### TASK-7.21 错误与 usage 审计字段

状态：done

范围：

1. 已区分安全展示文案和内部诊断详情：`error_message` 只保存 safe message，`internal_error_detail` 保存截断后的内部错误文本。
2. usage source 已区分非流式 `upstream_response` 和流式 final usage `upstream_stream`；manual_adjustment 等后台调整来源留到后续后台/人工调整能力。
3. 后续后台/console 请求日志默认只能展示 `error_message`；`internal_error_detail` 仅供内部 admin/排障权限查看。

关联 GAP：

- [GAP-7-005](../../production/TODO_REGISTER.md#gap-7-005) 已关闭
- [GAP-7-008](../../production/TODO_REGISTER.md#gap-7-008) 已关闭


<a id="task-7-22-price-effective-window"></a>
### TASK-7.22 价格生效窗口约束

状态：todo

范围：

1. 防止同一 model/currency/pricing_unit 出现重叠 enabled 生效窗口。
2. 后台改价时事务化关闭旧价格并启用新价格。
3. 结算时价格选择确定且可审计。

关联 GAP：

- [GAP-7-010](../../production/TODO_REGISTER.md#gap-7-010)
