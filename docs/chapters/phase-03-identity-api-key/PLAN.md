# Phase 3 Plan - 用户与 API Key

## 目标

建立 customer API 的身份入口：user、project、API key、认证 middleware 和基础限流。

本阶段的商业边界：

1. user 是当前个人账户模式下的付费主体。
2. project 是用户下的应用空间，用于组织 API key、归集用量和未来预算。
3. API key 是客户调用 OpenAI-compatible API 的身份凭据。
4. API key 不能保存明文，只能保存 hash 和 prefix。
5. 限流是保护平台的基础能力，但不能成为单点故障。

## 涉及文件

| 文件 | 作用 |
| --- | --- |
| [migrations/000002_create_users.up.sql](../../../migrations/000002_create_users.up.sql) | user schema。 |
| [migrations/000003_create_projects.up.sql](../../../migrations/000003_create_projects.up.sql) | project schema。 |
| [migrations/000004_create_api_keys.up.sql](../../../migrations/000004_create_api_keys.up.sql) | API key schema。 |
| [sql/queries/users.sql](../../../sql/queries/users.sql) | user 相关 SQL。 |
| [sql/queries/projects.sql](../../../sql/queries/projects.sql) | project 相关 SQL。 |
| [sql/queries/api_keys.sql](../../../sql/queries/api_keys.sql) | API key 相关 SQL。 |
| [internal/apikey](../../../internal/apikey) | API key 生成和创建服务。 |
| [internal/auth](../../../internal/auth) | API key 认证和 request auth context。 |
| [internal/middleware/api_key_auth.go](../../../internal/middleware/api_key_auth.go) | HTTP API key auth middleware。 |
| [internal/ratelimit](../../../internal/ratelimit) | 限流器和 Redis store。 |
| [internal/middleware/rate_limit.go](../../../internal/middleware/rate_limit.go) | HTTP 限流 middleware。 |

## 任务

<a id="task-3-01-identity-schema"></a>
### TASK-3.01 User、project、API key schema

状态：done

目标：

```text
建立客户身份、应用空间和 API key 的数据库事实。
```

已完成：

1. user 表。
2. project 表。
3. api_key 表。
4. API key prefix/hash 字段。
5. project 到 user 的归属关系。

关键约束：

1. API key 创建后只展示一次完整明文。
2. 数据库不能保存完整明文 API key。
3. API key 属于 project，不直接承载余额。
4. request 进入 gateway 后必须能追踪 user、project、api_key。

<a id="task-3-02-api-key-auth"></a>
### TASK-3.02 API key 认证 middleware

状态：partial

目标：

```text
把 HTTP 请求中的 opaque API key 转换成内部 AuthContext。
```

已完成：

1. 从 Authorization header 读取 bearer key。
2. 对 key 做 hash 后查询数据库。
3. 验证 key 是否存在、是否可用。
4. 将 user_id、project_id、api_key_id 放入 context。

当前欠账：

```text
每次认证同步更新 last_used_at，未来高流量下会放大数据库写入。
```

计划实现：

1. 评估节流更新，例如同一个 key 每 N 分钟最多更新一次。
2. 评估异步批量更新。
3. 后台展示 last used 时接受分钟级延迟。

关联 GAP：

- [GAP-3-001](../../production/TODO_REGISTER.md#gap-3-001)


<a id="task-3-03-api-key-management"></a>
### TASK-3.03 API Key 管理、禁用与审计

状态：partial

目标：

```text
让 API key 能被安全创建、查询、禁用、撤销，并能审计是谁操作。
```

已完成：

1. 创建 key 时传入 authenticated principal。
2. 校验 principal 是否拥有目标 project。
3. 创建成功只返回一次明文 key。

剩余计划：

1. 支持 list key，只返回 prefix、name、状态、创建时间和最后使用时间。
2. 支持 revoke/disable。
3. 后台敏感操作写 audit log。
4. 后续进入后台 admin 权限后，再扩展 admin principal 跨用户管理能力。

涉及文件：

1. [internal/apikey/service.go](../../../internal/apikey/service.go)
2. [internal/auth/apikey.go](../../../internal/auth/apikey.go)
3. 未来后台 handler。

关联 GAP：

- [GAP-3-002](../../production/TODO_REGISTER.md#gap-3-002)
- [GAP-3-007](../../production/TODO_REGISTER.md#gap-3-007)


<a id="task-3-04-rate-limit-production"></a>
### TASK-3.04 限流生产化

状态：done

目标：

```text
让基础限流既能保护平台，又不会因为 Redis 故障把客户请求全部打死。
```

已完成：

1. 将默认限流窗口和阈值迁入 config。
2. Redis key 使用统一 namespace。
3. `.env.example` 已登记默认限流和 Redis key namespace 配置。
4. Redis 限流计数使用 `TxPipelined + ExpireNX + TTL`，避免 `INCR` 成功但过期时间设置失败留下永久 key。
5. Redis 限流故障策略支持 `fail_closed` / `fail_open` 配置。
6. 降级路径记录脱敏 structured log，只记录 API key prefix，不记录完整 key。

后续能力：

1. Prometheus metrics 在阶段 8 [TASK-8.02](../phase-08-observability-stability/PLAN.md#task-8-02-metrics) 统一实现。
2. project/model/channel 级限流策略在后台策略能力进入后再扩展。

涉及文件：

1. [cmd/server/main.go](../../../cmd/server/main.go)
2. [internal/ratelimit/redis_store.go](../../../internal/ratelimit/redis_store.go)
3. [internal/middleware/rate_limit.go](../../../internal/middleware/rate_limit.go)

验证方式：

```bash
go test ./internal/ratelimit ./internal/middleware
```

关联 GAP：

- [GAP-3-003](../../production/TODO_REGISTER.md#gap-3-003)
- [GAP-3-004](../../production/TODO_REGISTER.md#gap-3-004)
- [GAP-3-005](../../production/TODO_REGISTER.md#gap-3-005)
- [GAP-3-006](../../production/TODO_REGISTER.md#gap-3-006)
