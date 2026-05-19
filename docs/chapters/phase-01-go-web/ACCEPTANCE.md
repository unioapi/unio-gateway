# Phase 1 Acceptance

## 功能验收

1. 服务可以启动和优雅退出。
2. `/healthz` 返回稳定 JSON 响应。
3. HTTP router、handler、httpx 工具边界清楚。
4. 业务层不依赖 chi。

## 生产验收

1. HTTP server timeout 全部可配置。
2. graceful shutdown timeout 可配置。
3. readiness 与 liveness 语义清楚。
4. `X-Request-ID` 输入有长度和字符集限制。

## 测试验收

1. router 基础测试通过。
2. health handler 测试通过。
3. request id middleware 覆盖客户端传入、缺失和非法值场景。

## 文档验收

1. 所有阶段 1 production TODO 都登记到 TODO register。
2. 阶段 1 状态文件能区分已完成和欠账。

