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

```text
GAP-7-009
GAP-7-010
```

<a id="task-7-02-request-attempt-record"></a>
### TASK-7.02 Request record 与 attempt record

状态：partial

范围：

1. 创建 request record。
2. 创建 attempt record。
3. 记录 user/project/api_key/model/provider/channel。
4. 记录 succeeded/failed/canceled 状态。
5. 后续补状态机守卫。

关联 GAP：

```text
GAP-7-003
GAP-7-005
```

<a id="task-7-03-usage-record"></a>
### TASK-7.03 Usage record

状态：partial

范围：

1. 记录 prompt/completion/total token。
2. 记录 cached/reasoning token。
3. 后续区分非流式 response 与 stream final usage 来源。

关联 GAP：

```text
GAP-7-008
```

<a id="task-7-04-ledger-debit"></a>
### TASK-7.04 Ledger debit

状态：partial

范围：

1. 支持 credit/debit ledger entry。
2. debit 使用事务更新 user balance。
3. 后续补 pre-authorize/capture/refund/reservation。

关联 GAP：

```text
GAP-7-011
GAP-7-012
```

<a id="task-7-05-non-stream-settlement"></a>
### TASK-7.05 非流式 settlement

状态：partial

范围：

1. adapter 返回 usage 后创建 usage record。
2. 创建 price snapshot。
3. 创建 ledger debit。
4. 标记 request/attempt succeeded。
5. 后续补余额预检、预授权和 settlement 幂等。

关联 GAP：

```text
GAP-7-001
GAP-7-007
GAP-7-012
```

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
3. 无 final usage 时不强行估算扣费。
4. 后续补预授权、异常策略和风控。

关联 GAP：

```text
GAP-7-002
GAP-7-004
GAP-7-006
GAP-7-011
```

## 当前必须收口任务

<a id="task-7-17-preauthorization"></a>
### TASK-7.17 余额预检查与预授权最小闭环

状态：todo

范围：

1. 请求调用上游前检查用户余额。
2. 非流式请求支持余额 preflight 或 reservation。
3. 流式请求根据模型价格和 `max_tokens` 做预授权。
4. 成功后按真实 usage capture。
5. 失败、取消、无 final usage 时 release/refund 或进入异常状态。
6. 所有余额动作都必须可审计。

生产风险：

```text
没有预授权时，余额不足用户可能先消耗上游成本，再在结算阶段失败。
stream 长输出和恶意断开也无法控损。
```

关联 GAP：

```text
GAP-7-001
GAP-7-002
GAP-7-004
GAP-7-011
```

<a id="task-7-18-request-state-machine"></a>
### TASK-7.18 Request/attempt 状态机守卫

状态：todo

范围：

1. 明确 pending、succeeded、failed、canceled 等状态转移。
2. SQL 更新必须带合法前置状态条件。
3. 终态不能被补偿任务或并发重试覆盖。
4. 重复终态更新应具备幂等语义。

关联 GAP：

```text
GAP-7-003
```

<a id="task-7-19-settlement-idempotency"></a>
### TASK-7.19 Settlement 幂等

状态：todo

范围：

1. 以 request_record_id 作为 settlement 幂等边界。
2. 重复 settlement 时检测既有 usage/snapshot/ledger。
3. 并发 settlement 不能把已成功请求误标失败。
4. 外部事务内的 ledger 幂等冲突需要稳定重入策略。

关联 GAP：

```text
GAP-7-007
GAP-7-012
```

<a id="task-7-20-cost-snapshot"></a>
### TASK-7.20 成本价与毛利审计

状态：todo

范围：

1. 区分客户侧售卖价和 provider/channel 成本价。
2. request settlement 保存 cost snapshot。
3. 支持按 channel/provider 分析毛利和 fallback 成本。

关联 GAP：

```text
GAP-7-009
```

<a id="task-7-21-error-usage-audit"></a>
### TASK-7.21 错误与 usage 审计字段

状态：todo

范围：

1. 区分 safe_user_message 和 internal_error_detail。
2. usage source 区分 response、stream_final_usage、manual_adjustment 等。
3. 后台展示前统一脱敏。

关联 GAP：

```text
GAP-7-005
GAP-7-008
```

<a id="task-7-22-price-effective-window"></a>
### TASK-7.22 价格生效窗口约束

状态：todo

范围：

1. 防止同一 model/currency/pricing_unit 出现重叠 enabled 生效窗口。
2. 后台改价时事务化关闭旧价格并启用新价格。
3. 结算时价格选择确定且可审计。

关联 GAP：

```text
GAP-7-010
```
