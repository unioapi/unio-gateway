# Phase 8 Plan - 可观测性与稳定性

## 目标

让平台具备生产排障、稳定性治理和 provider/channel 健康管理能力。

阶段 8 不是简单加日志，而是要让系统回答这些生产问题：

1. 哪个 project、model、provider、channel 的错误率最高？
2. 是用户请求错误、平台配置错误、上游认证错误、上游限流，还是上游 5xx？
3. 哪些 channel 应该被降权、熔断或临时禁用？
4. stream 中断发生在写出前还是写出后？
5. request、attempt、usage、ledger 能否通过 correlation id 串起来？

进入阶段 8 前，阶段 7 的 P0 计费阻断必须先关闭。

## 涉及文件

| 文件 | 作用 |
| --- | --- |
| [internal/gateway](../../../internal/gateway) | request 编排、attempt 状态、fallback 决策。 |
| [internal/adapter](../../../internal/adapter) | provider error 和 upstream metadata 的来源。 |
| [internal/middleware/logger.go](../../../internal/middleware/logger.go) | HTTP structured log。 |
| [internal/middleware/request_id.go](../../../internal/middleware/request_id.go) | correlation id。 |
| [internal/routing/router.go](../../../internal/routing/router.go) | channel 选择和后续 health 策略入口。 |
| [docs/production/THIRD_PARTY_POLICY.md](../../production/THIRD_PARTY_POLICY.md) | Prometheus/OpenTelemetry 等依赖选择规则。 |

## 任务

<a id="task-8-01-adapter-metadata-provider-errors"></a>
### TASK-8.01 Adapter metadata 与 provider error classification

状态：planned

目标：

```text
让 adapter 不只返回 error string，而是返回 gateway 可以理解和决策的结构化错误。
```

计划实现：

1. adapter response metadata 增加 upstream status code。
2. adapter response metadata 增加 upstream request id。
3. adapter error 增加 category。
4. category 至少区分 auth、permission、rate_limit、timeout、canceled、bad_request、server_error、unknown。
5. gateway 根据 category 决定是否 retry/fallback。
6. HTTP 层继续返回安全错误，不直接透传上游 body。

关键约束：

1. 上游 `401/403` 多数是平台 channel credential 问题，不应该让用户误以为自己的 API key 错了。
2. stream 已经写出后不能跨 channel fallback。
3. provider 原始错误只进入内部日志和 request record 内部字段。

关联 GAP：

- [GAP-8-001](../../production/TODO_REGISTER.md#gap-8-001)


<a id="task-8-02-metrics"></a>
### TASK-8.02 Prometheus metrics

状态：planned

目标：

```text
提供可聚合、低基数、可告警的业务和基础设施 metrics。
```

计划实现：

1. 引入 Prometheus 官方 client。
2. HTTP request count、latency、status。
3. Gateway request count、success/failure/canceled。
4. Adapter upstream latency、error category。
5. Routing selected provider/channel/model count。
6. Billing settlement success/failure count。
7. Stream started/completed/canceled/missing_usage count。
8. Rate limit decision、limited、Redis failure 和 fail-open count。

label 约束：

1. 可以使用 project_id、model、provider、channel、status、error_category。
2. 不要使用 raw request_id、api key、user input、完整 URL 作为 label。
3. API key 只能用 prefix 或 internal id，且默认不放高基数 label。

验证方式：

```bash
go test ./internal/...
curl /metrics
```

<a id="task-8-03-logs-traces"></a>
### TASK-8.03 Structured logs 与 OpenTelemetry

状态：planned

目标：

```text
让一次请求能从 HTTP、gateway、routing、adapter、billing 串起来。
```

计划实现：

1. 统一 log fields：correlation_id、request_id、user_id、project_id、api_key_id、model、provider、channel。
2. 敏感字段脱敏：API key、credential、上游 Authorization header、用户 prompt 默认不进日志。
3. 阶段 8 后期接入 OpenTelemetry trace。
4. trace span 按 HTTP、gateway、routing、adapter、settlement 拆分。
5. request record 的业务 request_id 与 HTTP correlation id 都要可查。

常见坑：

1. 不要把用户 prompt 全量打进日志。
2. 不要把 credential_ref 解析后的明文 credential 打进日志。
3. 不要让高基数字段污染 metrics label。

<a id="task-8-04-retry-circuit-breaker"></a>
### TASK-8.04 Retry 与 circuit breaker

状态：planned

目标：

```text
让 fallback 和 channel health 基于明确错误分类，而不是靠字符串猜测。
```

计划实现：

1. 先完成 provider error classification。
2. 定义 retryable error。
3. 非流式请求在未写出响应前可以尝试同模型 channel fallback。
4. stream 在写出后不能 fallback。
5. channel health 根据错误率和时间窗口降权或熔断。
6. routing 查询排除熔断 channel。
7. 后台可以查看和手动恢复 channel health。

完成标准：

1. 401/403 不盲目重试。
2. timeout/5xx 可以按策略 retry/fallback。
3. 429 可以按策略降权。
4. 所有 retry/fallback 都写 attempt record。
