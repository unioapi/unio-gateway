# Phase 5 Plan - Adapter 边界

## 目标

建立 Unio 内部 adapter contract，使 gateway 不依赖 provider-specific HTTP 细节。

adapter 只负责协议转换、请求发送、响应解析、stream parser、usage/error 映射，不负责 routing、credential 存储、价格和计费。

## 任务

<a id="task-5-01-chat-parameter-contract"></a>
### TASK-5.01 Chat 参数 contract

状态：todo

范围：

1. 盘点 HTTP DTO 已接收的 OpenAI-compatible 参数。
2. 明确每个参数进入 `adapter.ChatRequest` 还是被 HTTP 层显式拒绝。
3. OpenAI adapter wire DTO 完整承载已支持参数。
4. 防止用户传参被静默丢弃。

关联 GAP：

```text
GAP-5-001
```

<a id="task-5-02-openai-non-stream-adapter"></a>
### TASK-5.02 OpenAI 非流式 adapter

状态：done

范围：

1. 转换内部 chat request 到 OpenAI wire request。
2. 调用上游 HTTP。
3. 解析 OpenAI response。
4. 映射 prompt/completion/total/cached/reasoning usage。

<a id="task-5-03-openai-stream-adapter"></a>
### TASK-5.03 OpenAI stream adapter

状态：partial

范围：

1. 发送 stream request。
2. 解析 SSE delta。
3. 映射 final usage chunk。
4. 后续替换 `bufio.Scanner` parser。

关联 GAP：

```text
GAP-5-002
```

<a id="task-5-04-adapter-error-metadata"></a>
### TASK-5.04 Adapter 错误与 metadata contract

状态：planned

范围：

1. adapter 返回结构化 provider error。
2. adapter 暴露 upstream status、upstream request id。
3. gateway 根据错误分类决定 fallback、记录和用户可见错误。

关联 GAP：

```text
GAP-8-001
```

