# Phase 4 Plan - OpenAI-compatible API

## 目标

对外提供 OpenAI-compatible HTTP API 入口。

HTTP 层只负责协议入口、DTO decode/validation、错误格式和 SSE 写出，不负责 routing、provider/channel 选择和计费。

## 任务

<a id="task-4-01-models-endpoint"></a>
### TASK-4.01 `/v1/models`

状态：partial

范围：

1. 提供 OpenAI-compatible models response。
2. 从 model catalog 读取可用模型。
3. 后续与 project 级可见性策略对齐。

关联 GAP：

```text
GAP-6-006
```

<a id="task-4-02-chat-endpoint"></a>
### TASK-4.02 `/v1/chat/completions`

状态：done

范围：

1. 接收 OpenAI-compatible chat request。
2. 区分非流式与流式请求。
3. 调用 gateway。
4. 返回 OpenAI-compatible response 或 SSE。

<a id="task-4-03-chat-dto-validation"></a>
### TASK-4.03 Chat DTO 深度校验

状态：todo

范围：

1. 校验 message role 合法性。
2. 明确 content 空值策略。
3. 校验 stop、user、temperature、top_p、max_tokens 等字段边界。
4. 对暂不支持字段选择显式拒绝或完整转发。

关联 GAP：

```text
GAP-4-001
GAP-5-001
```

<a id="task-4-04-strict-json-error"></a>
### TASK-4.04 严格 JSON decode 与错误格式

状态：todo

范围：

1. 校验 Content-Type。
2. 拒绝尾随 JSON token。
3. 区分 body too large、malformed JSON、unknown field 等错误。
4. 统一映射成安全的 OpenAI-compatible error。

关联 GAP：

```text
GAP-4-002
```

<a id="task-4-05-sse-write-error"></a>
### TASK-4.05 SSE 写出后的错误语义

状态：partial

范围：

1. SSE 写出前可以返回 JSON error。
2. SSE 写出后不能再切换为 JSON error。
3. 写出后错误必须依赖 request 状态、日志、metrics 和后续错误事件/观测能力表达。

关联 GAP：

```text
GAP-7-006
```
