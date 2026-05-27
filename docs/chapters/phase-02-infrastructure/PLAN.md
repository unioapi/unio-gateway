# Phase 2 Plan - 基础设施

## 目标

建立 Unio API 的进程级基础设施：配置、PostgreSQL、Redis、migrations 和 sqlc。

本阶段的重点不是“能连上数据库”，而是确定基础设施和业务数据的边界：

1. config 只放启动配置和基础设施配置。
2. provider/channel/model/price 不进入正式 env/config。
3. PostgreSQL 是账务和业务事实来源。
4. Redis 只做缓存、限流或临时状态，不做金额事实来源。
5. migrations 和 sqlc 必须让 schema 变化可审查、可回滚、可测试。

## 涉及文件

| 文件 | 作用 |
| --- | --- |
| [internal/config/config.go](../../../internal/config/config.go) | 进程配置加载和默认值。 |
| [internal/store/postgres.go](../../../internal/store/postgres.go) | PostgreSQL pool 创建和健康检查入口。 |
| [internal/redis/client.go](../../../internal/redis/client.go) | Redis client 创建入口。 |
| [migrations](../../../migrations) | 数据库结构迁移。 |
| [sql/queries](../../../sql/queries) | sqlc 查询定义。 |
| [internal/store/sqlc](../../../internal/store/sqlc) | sqlc 生成代码和数据库测试。 |

## 任务

<a id="task-2-01-config-boundary"></a>
### TASK-2.01 Config 边界

状态：done

目标：

```text
让 config 只表达进程启动和基础设施，不承载可后台管理的业务数据。
```

已完成：

1. 基础 config 结构已经存在。
2. PostgreSQL DSN、Redis 地址等基础配置已进入 config。
3. provider/channel/model 已进入数据库业务数据，当前正式 config 未承载 provider/channel env。

当前欠账：

1. 无。

设计约束：

1. `OPENAI_API_KEY` 可以作为开发期 seed/playground，但不能成为正式 provider credential 来源。
2. `provider`、`channel`、`model`、`price` 属于业务数据，最终由数据库和后台管理。
3. KMS/master key、全局默认 timeout、部署开关属于 config。

关联 GAP：

- [GAP-2-003](../../production/TODO_REGISTER.md#gap-2-003)
- [GAP-2-004](../../production/TODO_REGISTER.md#gap-2-004)
- [GAP-6-004](../../production/TODO_REGISTER.md#gap-6-004)


<a id="task-2-02-postgres-pool"></a>
### TASK-2.02 PostgreSQL pool 生产参数

状态：done

目标：

```text
让 PostgreSQL 连接池在生产环境可控，避免默认参数在高并发或故障时放大问题。
```

已完成：

1. 在 config 中增加 `MaxConns`、`MinConns`。
2. 增加 `MaxConnLifetime`、`MaxConnIdleTime`。
3. 增加连接池 health check period。
4. 在 [internal/store/postgres.go](../../../internal/store/postgres.go) 中使用 `pgxpool.ParseConfig` 配置 pool。
5. 启动阶段执行轻量 ping，失败时阻止服务启动。

验证方式：

```bash
go test ./internal/config ./internal/store
go test ./internal/store/sqlc
```

关联 GAP：

- [GAP-2-003](../../production/TODO_REGISTER.md#gap-2-003)
- [GAP-2-005](../../production/TODO_REGISTER.md#gap-2-005)


<a id="task-2-03-redis-pool-namespace"></a>
### TASK-2.03 Redis timeout、pool 与 namespace

状态：done

目标：

```text
让 Redis 作为辅助基础设施时具备可配置的超时、连接池和 key 隔离。
```

已完成：

1. 在 config 中增加 Redis dial/read/write timeout。
2. 增加 Redis pool size。
3. 增加 Redis retry 参数。
4. 在 [internal/redis/client.go](../../../internal/redis/client.go) 中使用 config 注入 Redis client 连接参数。
5. 在 `.env.example` 中登记 Redis client 可配置项。
6. 增加全局 key namespace，例如 `unio:dev`、`unio:prod`。
7. rate limit Redis key 已统一拼接 namespace。
8. Redis 限流故障降级策略已在阶段 3 [GAP-3-006](../../production/TODO_REGISTER.md#gap-3-006) 完成。

涉及文件：

1. [internal/config/config.go](../../../internal/config/config.go)
2. [internal/redis/client.go](../../../internal/redis/client.go)
3. [internal/ratelimit/redis_store.go](../../../internal/ratelimit/redis_store.go)

关联 GAP：

- [GAP-2-004](../../production/TODO_REGISTER.md#gap-2-004)
- [GAP-2-007](../../production/TODO_REGISTER.md#gap-2-007)
- [GAP-3-005](../../production/TODO_REGISTER.md#gap-3-005)


<a id="task-2-04-migration-runner"></a>
### TASK-2.04 Migration runner 与 schema version

状态：todo

目标：

```text
服务启动时能确定数据库 schema 与代码匹配，避免连接到未迁移或版本错误的数据库。
```

计划实现：

1. 按 [THIRD_PARTY_POLICY.md](../../production/THIRD_PARTY_POLICY.md) 评估 migration runner。
2. 引入正式 migration runner 或 schema version checker。
3. 启动前检查当前 schema version。
4. version 不匹配时拒绝启动。
5. 决定 [000001_create_schema_health_checks.up.sql](../../../migrations/000001_create_schema_health_checks.up.sql) 的长期定位。

常见坑：

1. 不要让服务在 schema 不匹配时“先启动再说”。
2. 不要用业务表存在与否代替 migration version。
3. 不要把本地开发的自动迁移策略直接带到生产。

关联 GAP：

- [GAP-2-001](../../production/TODO_REGISTER.md#gap-2-001)
- [GAP-2-002](../../production/TODO_REGISTER.md#gap-2-002)
- [GAP-2-006](../../production/TODO_REGISTER.md#gap-2-006)


<a id="task-2-05-updated-at-strategy"></a>
### TASK-2.05 updated_at 策略

状态：todo

目标：

```text
统一所有表的 updated_at 更新语义，避免后续审计和后台展示出现不一致。
```

计划实现：

1. 在 trigger 和显式 SQL 更新之间选择一种长期策略。
2. 如果选择显式 SQL，则每个 update query 必须维护 `updated_at`。
3. 如果选择 trigger，则 migration 中统一创建 trigger function。
4. 为关键表补更新测试。

涉及文件：

1. [migrations/000002_create_users.up.sql](../../../migrations/000002_create_users.up.sql)
2. [sql/queries](../../../sql/queries)

关联 GAP：

- [GAP-2-008](../../production/TODO_REGISTER.md#gap-2-008)
