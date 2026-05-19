# Phase 5 Plan - Adapter 边界

## 目标

建立 Unio 内部 adapter contract，让 gateway 不依赖 provider-specific HTTP 细节。

adapter 是代码能力，不是 provider/channel 管理系统。它只负责：

1. 请求转换。
2. 上游 HTTP 调用。
3. 响应解析。
4. stream parser。
5. usage 映射。
6. provider-specific error/metadata 映射。

adapter 不负责：

1. 选择 provider/channel。
2. 读取正式 provider env。
3. 查询数据库。
4. 管理 credential。
5. 计费和 ledger。

## 涉及文件

| 文件 | 作用 |
| --- | --- |
| [internal/adapter/chat.go](../../../internal/adapter/chat.go) | 内部 chat adapter contract。 |
| [internal/adapter/registry.go](../../../internal/adapter/registry.go) | adapter registry。 |
| [internal/adapter/openai/dto.go](../../../internal/adapter/openai/dto.go) | OpenAI 上游 wire DTO。 |
| [internal/adapter/openai/chat.go](../../../internal/adapter/openai/chat.go) | OpenAI adapter 实现。 |
| [internal/channel/runtime.go](../../../internal/channel/runtime.go) | gateway/routing 传给 adapter 的运行时 channel 参数。 |

## 任务

<a id="task-5-01-chat-parameter-contract"></a>
### TASK-5.01 Chat 参数 contract

状态：todo

目标：

```text
避免用户传入的 OpenAI-compatible 参数在 HTTP 层被接收，却在 adapter 层被静默丢弃。
```

当前欠账：

1. HTTP DTO 已经接收部分参数。
2. `adapter.ChatRequest` 没有完整承载这些字段。
3. OpenAI wire DTO 也没有完整映射所有已支持字段。

计划实现：

1. 盘点 [internal/httpapi/openai_dto.go](../../../internal/httpapi/openai_dto.go) 中所有 chat request 字段。
2. 将支持字段加入 [internal/adapter/chat.go](../../../internal/adapter/chat.go)。
3. 将支持字段映射到 [internal/adapter/openai/dto.go](../../../internal/adapter/openai/dto.go)。
4. 对不支持字段在 HTTP validation 阶段显式拒绝。
5. 为每个字段写清楚谁创建、谁消费、何时使用。

完成标准：

1. 已支持参数不会丢。
2. 不支持参数不会假装支持。
3. optional scalar 使用指针保留显式零值。

关联 GAP：

- [GAP-5-001](../../production/TODO_REGISTER.md#gap-5-001)


<a id="task-5-02-openai-non-stream-adapter"></a>
### TASK-5.02 OpenAI 非流式 adapter

状态：done

目标：

```text
把内部 chat request 转换为 OpenAI-compatible 上游请求，并解析非流式响应。
```

已完成：

1. 构造 OpenAI request。
2. 使用 `channel.Runtime` 中的 base URL、credential、model mapping。
3. 发起 HTTP 请求。
4. 解析 OpenAI response。
5. 映射 `prompt_tokens`、`completion_tokens`、`total_tokens`。
6. 映射 `cached_tokens`。
7. 映射 `reasoning_tokens`。

关键约束：

1. adapter 不知道用户余额。
2. adapter 不写 request record。
3. adapter 不读取 database。

<a id="task-5-03-openai-stream-adapter"></a>
### TASK-5.03 OpenAI stream adapter

状态：partial

目标：

```text
支持 OpenAI-compatible SSE stream，并能解析 final usage chunk。
```

已完成：

1. stream request 设置 `stream_options.include_usage=true`。
2. 解析普通 delta chunk。
3. 解析 usage-only final chunk。
4. 将 final usage 放入 `adapter.ChatStreamChunk.Usage`。

当前欠账：

```text
stream parser 仍基于 bufio.Scanner，受单个 event 大小限制。
```

计划实现：

1. 评估成熟 SSE parser，遵循 [THIRD_PARTY_POLICY.md](../../production/THIRD_PARTY_POLICY.md)。
2. 如果继续自研，改为基于 reader 的 event parser。
3. 明确 max event size。
4. 显式处理 `[DONE]`、空行、多行 data、异常 JSON。
5. 为 tool_calls、大 chunk、backpressure 增加测试。

关联 GAP：

- [GAP-5-002](../../production/TODO_REGISTER.md#gap-5-002)


<a id="task-5-04-adapter-error-metadata"></a>
### TASK-5.04 Adapter 错误与 metadata contract

状态：planned

目标：

```text
让 adapter 返回足够的 provider metadata，支撑阶段 8 的 error classification、fallback 和观测。
```

计划实现：

1. adapter error 结构化，区分 auth、rate limit、timeout、server error、bad request。
2. adapter response 暴露 upstream status code。
3. adapter response 暴露 upstream request id。
4. gateway 根据错误分类决定是否 retry/fallback。
5. 用户错误仍由 HTTP 层做安全映射，不透传上游原始 body。

关联 GAP：

- [GAP-8-001](../../production/TODO_REGISTER.md#gap-8-001)


