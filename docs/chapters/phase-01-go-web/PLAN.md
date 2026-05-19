# Phase 1 Plan - Go Web 骨架

## 目标

建立最小但边界清楚的 Go Web 服务骨架。

本阶段只处理进程启动、HTTP router、基础 middleware、健康检查、日志和优雅退出，不引入业务 provider/channel 逻辑。

## 任务

<a id="task-1-01-web-skeleton"></a>
### TASK-1.01 Web 服务骨架

状态：done

范围：

1. 创建 `cmd/server/main.go`。
2. 使用 `chi` 作为 HTTP router。
3. 接入 `slog`。
4. 提供 `/healthz`。
5. 建立 `internal/httpapi` 和 `internal/httpx` 边界。

关键约束：

```text
chi 只允许停留在 HTTP 层，业务层不接收 router/framework context。
```

<a id="task-1-02-server-timeouts-readiness"></a>
### TASK-1.02 Server timeout 与 readiness

状态：todo

范围：

1. 将 HTTP server read/write/idle timeout 纳入 config。
2. 将 graceful shutdown timeout 纳入 config。
3. 区分 liveness 与 readiness。
4. 后续把 readiness 状态暴露给部署和观测系统。

生产风险：

```text
公网 API 没有 timeout/readiness 会导致慢连接、部署滚动和故障切流不可控。
```

关联 GAP：

```text
GAP-1-001
```

<a id="task-1-03-correlation-id"></a>
### TASK-1.03 Correlation ID 输入约束

状态：todo

范围：

1. 接收客户端 `X-Request-ID` 时限制长度和字符集。
2. 非法值忽略并生成服务端 correlation id。
3. 响应头和日志只写入安全值。

生产风险：

```text
直接信任客户端 header 会污染日志、响应头和后续 tracing。
```

关联 GAP：

```text
GAP-1-002
```

