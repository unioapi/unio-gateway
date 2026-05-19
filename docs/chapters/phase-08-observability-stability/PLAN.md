# Phase 8 Plan - 可观测性与稳定性

## 目标

让平台具备生产排障、稳定性治理和 provider/channel 健康管理能力。

本阶段不改变阶段 7 的账务事实，但要让日志、metrics、traces、retry、fallback 和 provider error classification 可审计。

## 任务

<a id="task-8-01-adapter-metadata-provider-errors"></a>
### TASK-8.01 Adapter metadata 与 provider error classification

状态：planned

范围：

1. adapter response 暴露 upstream status code。
2. adapter response 暴露 upstream request id。
3. adapter error 映射成结构化分类。
4. gateway 根据错误分类决定 retry/fallback/用户错误。

关联 GAP：

```text
GAP-8-001
```

<a id="task-8-02-metrics"></a>
### TASK-8.02 Prometheus metrics

状态：planned

范围：

1. 请求数量、延迟、错误率。
2. 按 project/model/provider/channel 聚合。
3. 脱敏 API key 和 credential。

<a id="task-8-03-logs-traces"></a>
### TASK-8.03 Structured logs 与 OpenTelemetry

状态：planned

范围：

1. 统一 log fields。
2. correlation id 贯穿 HTTP/gateway/adapter。
3. 后续接入 OpenTelemetry trace。

<a id="task-8-04-retry-circuit-breaker"></a>
### TASK-8.04 Retry 与 circuit breaker

状态：planned

范围：

1. 基于 provider error classification 判断 retryable。
2. channel health 和 circuit breaker 影响 routing。
3. stream 已写出后不跨 channel fallback。

