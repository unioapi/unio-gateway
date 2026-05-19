# Phase 4 Plan - OpenAI-compatible API

## 目标

对外提供 OpenAI-compatible HTTP API 入口，让用户可以用接近 OpenAI SDK 的方式调用 Unio API。

本阶段必须守住三条边界：

1. HTTP 层只负责协议入口、DTO decode/validation、错误格式和 SSE 写出。
2. gateway 负责业务编排，HTTP handler 不选择 provider/channel，不计费。
3. 对外 DTO、内部 adapter DTO、OpenAI 上游 wire DTO 必须分开。

## 涉及文件

| 文件 | 作用 |
| --- | --- |
| [internal/httpapi/openai_dto.go](../../../internal/httpapi/openai_dto.go) | 对外 OpenAI-compatible DTO。 |
| [internal/httpapi/chat_completions_handler.go](../../../internal/httpapi/chat_completions_handler.go) | `/v1/chat/completions` HTTP handler。 |
| [internal/httpapi/models_handler.go](../../../internal/httpapi/models_handler.go) | `/v1/models` HTTP handler。 |
| [internal/httpapi/router.go](../../../internal/httpapi/router.go) | HTTP route 装配。 |
| [internal/httpx/json.go](../../../internal/httpx/json.go) | JSON decode。 |
| [internal/httpx/response.go](../../../internal/httpx/response.go) | JSON/error response。 |
| [internal/gateway](../../../internal/gateway) | chat completions 的业务编排。 |

## 任务

<a id="task-4-01-models-endpoint"></a>
### TASK-4.01 `/v1/models`

状态：partial

目标：

```text
让客户端能获取当前项目可见且可路由的模型列表。
```

已完成：

1. `GET /v1/models` endpoint 已存在。
2. response 使用 OpenAI-compatible list structure。
3. 数据来源已从空列表推进到 model catalog。

当前欠账：

```text
当前模型列表还是全局可用视角，未体现 project 级模型可见性、预算、禁用或专属 channel 策略。
```

计划实现：

1. handler 从 auth context 获取 project_id。
2. model catalog 查询接收 project_id。
3. catalog 与 routing 共用 project policy。
4. 保证“可见模型”和“可路由模型”一致。

关联 GAP：

- [GAP-6-006](../../production/TODO_REGISTER.md#gap-6-006)


<a id="task-4-02-chat-endpoint"></a>
### TASK-4.02 `/v1/chat/completions`

状态：done

目标：

```text
提供非流式和流式 chat completions HTTP 入口。
```

已完成：

1. decode OpenAI-compatible request。
2. 根据 `stream` 分支调用 gateway 非流式或流式方法。
3. 非流式返回 JSON。
4. 流式返回 SSE。
5. handler 不直接调用 adapter。

关键约束：

1. handler 创建的是 `httpapi.ChatCompletionRequest`，不是 `adapter.ChatRequest`。
2. handler 不知道 provider/channel。
3. handler 不做 billing settlement。

<a id="task-4-03-chat-dto-validation"></a>
### TASK-4.03 Chat DTO 深度校验

状态：todo

目标：

```text
公开 API 前，明确哪些 OpenAI-compatible 参数被支持、哪些被拒绝、哪些会透传。
```

当前欠账：

1. message 只校验了非空列表。
2. role 合法性未完整校验。
3. content 空值策略未定义。
4. `temperature`、`top_p`、`max_tokens`、`stop`、`user` 等参数边界未完整校验。
5. 部分已接收字段没有进入 adapter contract，存在静默丢参。

计划实现：

1. 明确支持的 role 集合。
2. 明确 text-only MVP 是否拒绝 tool/function/multimodal content。
3. 对 optional scalar 使用指针保留显式零值。
4. 对暂不支持字段返回 OpenAI-compatible error。
5. 与 [TASK-5.01](../phase-05-adapter-boundary/PLAN.md#task-5-01-chat-parameter-contract) 同步推进，防止 HTTP 层接受但 adapter 丢弃。

验证方式：

```bash
go test ./internal/httpapi
```

关联 GAP：

- [GAP-4-001](../../production/TODO_REGISTER.md#gap-4-001)
- [GAP-5-001](../../production/TODO_REGISTER.md#gap-5-001)


<a id="task-4-04-strict-json-error"></a>
### TASK-4.04 严格 JSON decode 与错误格式

状态：todo

目标：

```text
让公网 API 对 malformed request body 的响应稳定、可预期、OpenAI-compatible。
```

计划实现：

1. 校验 `Content-Type`。
2. 拒绝尾随 JSON token。
3. 区分 empty body、malformed JSON、body too large。
4. 将 decode 错误映射成安全的 OpenAI-compatible error。
5. 不把 Go 原始 decode 错误直接暴露给用户。

涉及文件：

1. [internal/httpx/json.go](../../../internal/httpx/json.go)
2. [internal/httpx/response.go](../../../internal/httpx/response.go)
3. [internal/httpapi/chat_completions_handler.go](../../../internal/httpapi/chat_completions_handler.go)

关联 GAP：

- [GAP-4-002](../../production/TODO_REGISTER.md#gap-4-002)


<a id="task-4-05-sse-write-error"></a>
### TASK-4.05 SSE 写出后的错误语义

状态：partial

目标：

```text
明确 stream response 一旦写出后，错误不能再用普通 JSON error 表达。
```

已完成：

1. stream handler 已使用 SSE。
2. gateway 已记录 request/attempt 状态。
3. 有 final usage 时能作为账务事实 settlement。

当前欠账：

1. 写出后 adapter error 只能表现为 stream 中断。
2. 客户端无法从 JSON body 得到标准错误。
3. 后续需要依赖日志、request record、metrics 和可能的 SSE error event 做排障。

计划实现：

1. 写出前错误继续返回 OpenAI-compatible JSON error。
2. 写出后错误只更新 request/attempt 状态。
3. 阶段 8 接入 metrics/logs 后暴露 stream 中断原因。
4. 不做跨 channel fallback，因为已有 bytes 写给客户端。

关联 GAP：

- [GAP-7-006](../../production/TODO_REGISTER.md#gap-7-006)


