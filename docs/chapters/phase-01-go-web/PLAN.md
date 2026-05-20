# Phase 1 Plan - Go Web 骨架

## 目标

建立一个能长期承载商业 API 网关的 Go Web 进程骨架。

本阶段不是为了“能跑一个 HTTP server”这么简单，而是先确定几条后续不能破坏的边界：

1. HTTP framework 只停留在 HTTP 层。
2. 业务层只接收 `context.Context` 和明确的 DTO/domain struct。
3. 进程生命周期、日志、request correlation、health/readiness 从第一天就有清楚位置。
4. 未来接入 gateway、billing、routing 时不需要反推 Web 层结构。

## 涉及文件

| 文件 | 作用 |
| --- | --- |
| [cmd/server/main.go](../../../cmd/server/main.go) | 进程入口，负责配置加载、依赖装配、HTTP server 生命周期和退出信号。 |
| [internal/httpapi/router.go](../../../internal/httpapi/router.go) | HTTP router 组装，chi 只允许出现在这一层。 |
| [internal/httpx/response.go](../../../internal/httpx/response.go) | JSON/error response 基础工具。 |
| [internal/httpx/request_id.go](../../../internal/httpx/request_id.go) | 从 context 读取 correlation id 的 HTTP 辅助函数。 |
| [internal/middleware/request_id.go](../../../internal/middleware/request_id.go) | 请求 correlation id middleware。 |
| [internal/middleware/logger.go](../../../internal/middleware/logger.go) | HTTP structured log middleware。 |
| [internal/middleware/recoverer.go](../../../internal/middleware/recoverer.go) | panic recovery middleware。 |

## 任务

<a id="task-1-01-web-skeleton"></a>
### TASK-1.01 Web 服务骨架

状态：done

目标：

```text
让服务具备可启动、可关闭、可被测试的 HTTP 入口。
```

实现内容：

1. 创建 [cmd/server/main.go](../../../cmd/server/main.go)。
2. 使用 `chi` 创建 router，但不让 chi 类型进入业务 service。
3. 创建 `/healthz`。
4. 接入 `slog`。
5. 接入 panic recoverer。
6. 使用标准库 `http.Server`。
7. 用 `context.Context` 控制 graceful shutdown。

完成标准：

1. `go run ./cmd/server` 能启动。
2. `GET /healthz` 能返回成功响应。
3. 关闭进程时能走 graceful shutdown。
4. handler 测试不需要真实网络端口。

常见坑：

1. 不要在 service 方法里接收 `*http.Request` 或 chi context。
2. 不要把 router 初始化和业务对象构造混在大型匿名函数里。
3. 不要在 health handler 里顺手做复杂数据库检查，readiness 应该单独设计。

<a id="task-1-02-server-timeouts-readiness"></a>
### TASK-1.02 Server timeout 与 readiness

状态：partial

目标：

```text
让 HTTP server 在公网部署时具备慢连接保护、优雅下线和部署探针能力。
```

当前欠账：

```text
HTTP server timeout 和 shutdown timeout 已配置化。
startup timeout 仍硬编码，readiness 状态还没有独立。
```

已完成：

1. 在 config 中增加 HTTP server timeout 字段。
2. 明确 read、read header、write、idle timeout 默认值。
3. 将 graceful shutdown timeout 从硬编码迁入 config。

剩余计划：

4. 增加 readiness 状态，而不是只依赖 `/healthz`。
5. 后续 metrics 接入后，把 readiness 状态暴露给部署系统。

涉及文件：

1. [cmd/server/main.go](../../../cmd/server/main.go)
2. [internal/config/config.go](../../../internal/config/config.go)
3. [internal/httpapi/router.go](../../../internal/httpapi/router.go)

验证方式：

```bash
go test ./internal/config ./internal/httpapi
go test ./...
```

关联 GAP：

- [GAP-1-001](../../production/TODO_REGISTER.md#gap-1-001)


<a id="task-1-03-correlation-id"></a>
### TASK-1.03 Correlation ID 输入约束

状态：done

目标：

```text
让用户传入的 X-Request-ID 只能作为安全的日志关联 ID，不能污染响应头和日志。
```

实现内容：

```text
middleware 会校验客户端 X-Request-ID 的长度和字符集。
非法 header 会被忽略并替换为服务端生成的 correlation id。
日志字段、response header 和 context 中只保存清洗后的值。
```

已完成：

1. 限制 `X-Request-ID` 最大长度。
2. 限制字符集，只允许可打印且适合日志/响应头的字符。
3. 非法 header 直接忽略并生成服务端 correlation id。
4. 日志字段、response header 和 context 中只保存清洗后的值。
5. request record 的业务 `request_id` 继续由 `requestlog.GenerateRequestID` 生成，不能复用客户端 correlation id。

涉及文件：

1. [internal/middleware/request_id.go](../../../internal/middleware/request_id.go)
2. [internal/httpx/request_id.go](../../../internal/httpx/request_id.go)
3. [internal/requestlog/request_id.go](../../../internal/requestlog/request_id.go)

验证方式：

```bash
go test ./internal/middleware ./internal/httpx ./internal/requestlog
```

完成标准：

1. 缺失 `X-Request-ID` 时生成新 correlation id。
2. 合法 `X-Request-ID` 被保留。
3. 超长或含控制字符的 `X-Request-ID` 被替换。
4. response header 不包含非法值。

关联 GAP：

- [GAP-1-002](../../production/TODO_REGISTER.md#gap-1-002)
