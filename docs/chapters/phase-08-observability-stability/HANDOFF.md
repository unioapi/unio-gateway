# Phase 8 Handoff - Failure Foundation

更新时间：2026-05-21

## 当前状态

结构化 Failure 基础已经接入项目主链路。

已完成：

1. 新增 `internal/failure`，统一承载稳定 `Code`、`Category`、message、cause 和少量安全 fields。
2. `failure.Code` 使用 `<category>_<reason>` 格式，`Category` 从第一个 `_` 前缀推导。
3. `failure.LogArgs` 已用于启动错误日志和 rate limit 故障日志。
4. Config、HTTP JSON、Postgres、Redis、auth、apikey、ratelimit、routing、credential、modelcatalog、adapter、SSE、billing、ledger、requestlog 和 gateway 关键错误已接入 failure。
5. 原有 sentinel error 继续保留，并作为 `failure.Wrap` 的 cause，因此 `errors.Is` 判断仍然可用。
6. `request_records.error_code` 和 `request_attempts.error_code` 已优先使用 `failure.CodeOf(err)`。
7. 测试已调整为断言 `failure.CodeOf`、`failure.CategoryOf` 和 `errors.Is`，不再依赖完整错误字符串。

## 重要规范

以后新增错误时遵循：

1. 需要跨模块传播、日志记录、写入 request/attempt error_code、参与 retry/fallback 判断的错误，必须定义 `failure.Code`。
2. 模块内可以保留 `ErrXxx` sentinel，但返回给上层时要使用 `failure.Wrap(code, ErrXxx, ...)`。
3. `failure.WithMessage` 是内部诊断消息，不是用户可见文案。
4. `failure.WithField` 只放少量安全、确实需要检索的字段；不要为每个模块维护一套 Field 常量。
5. HTTP 层负责把内部 failure 映射成安全的 OpenAI-compatible error，不直接暴露 `err.Error()`。
6. 日志记录错误时用 `failure.LogArgs(err)`，不要手写分散的 `error_code`、`error_category`。
7. 测试优先断言稳定 code/category 和 cause，不断言完整错误字符串。

## 仍需注意

这次只是 Failure 基础，不等于阶段 8 完成。

仍未完成：

1. Provider 原始错误 body 的脱敏解析和 metadata contract。
2. retry/fallback 基于 provider 错误类型的精细分类。
3. channel health 根据错误率降权或熔断。
4. SSE Writer、heartbeat 和更完整的 stream observability。

相关 GAP：

1. [GAP-8-001](../../production/TODO_REGISTER.md#gap-8-001)
2. [GAP-8-002](../../production/TODO_REGISTER.md#gap-8-002)

说明：

```text
GAP-7-006 已在 2026-05-29 关闭：Chat Completions SSE 已开始后会写出 OpenAI-compatible data-only error chunk，并且不写 [DONE]。
阶段 8 后续只增强项目级 SSE Writer、metrics、日志和 observability，不再阻塞阶段 7。
```

## 建议阅读顺序

1. [internal/failure/code.go](../../../internal/failure/code.go)
2. [internal/failure/failure.go](../../../internal/failure/failure.go)
3. [internal/failure/log.go](../../../internal/failure/log.go)
4. [internal/config/config.go](../../../internal/config/config.go)
5. [internal/httpx/json.go](../../../internal/httpx/json.go)
6. [internal/routing/router.go](../../../internal/routing/router.go)
7. [internal/adapter/openai/chat.go](../../../internal/adapter/openai/chat.go)
8. [internal/gateway/chat_request_record.go](../../../internal/gateway/chat_request_record.go)

## 验证命令

```bash
go test ./...
```

最近一次验证：2026-05-21，通过。

手动启动错误日志验证：

```bash
REDIS_DB=abc go run ./cmd/server
```

期望看到：

```text
error="parse REDIS_DB as int" error_code=config_invalid error_category=config
```

## 下一步

如果继续阶段 8：

```text
TASK-8.01 adapter metadata 和 provider error classification
```

如果回到阶段 7 主线：

```text
7.17 余额预检查与预授权最小闭环
```
