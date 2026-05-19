# Phase 2 Plan - 基础设施

## 目标

建立项目运行所需的基础设施能力，包括 config、PostgreSQL、Redis、migrations 和 sqlc。

本阶段只处理进程级基础设施，不承载 provider/channel/model/price 业务数据。

## 任务

<a id="task-2-01-config-boundary"></a>
### TASK-2.01 Config 边界

状态：partial

范围：

1. 加载进程启动配置。
2. 区分基础设施 config 和业务数据。
3. PostgreSQL/Redis 连接配置进入 config。
4. provider/channel/model/price 不进入正式 config。

关联 GAP：

```text
GAP-2-003
GAP-2-004
GAP-6-004
```

<a id="task-2-02-postgres-pool"></a>
### TASK-2.02 PostgreSQL pool 生产参数

状态：todo

范围：

1. 配置 max conns、min conns、max conn lifetime、idle time。
2. 增加健康检查 timeout。
3. 启动时验证连接和 pool 参数。

关联 GAP：

```text
GAP-2-001
GAP-2-003
GAP-2-005
```

<a id="task-2-03-redis-pool-namespace"></a>
### TASK-2.03 Redis timeout、pool 与 namespace

状态：todo

范围：

1. 配置 Redis dial/read/write timeout。
2. 配置 pool size。
3. 定义 key namespace，隔离环境。
4. 明确 Redis 故障降级策略由调用方选择。

关联 GAP：

```text
GAP-2-004
GAP-2-007
GAP-3-005
```

<a id="task-2-04-migration-runner"></a>
### TASK-2.04 Migration runner 与 schema version

状态：todo

范围：

1. 引入正式 migration runner。
2. 启动前校验 schema 版本。
3. 决定开发期 schema health table 的长期定位。
4. schema 不匹配时阻止服务继续启动。

关联 GAP：

```text
GAP-2-001
GAP-2-002
GAP-2-006
```

<a id="task-2-05-updated-at-strategy"></a>
### TASK-2.05 updated_at 策略

状态：todo

范围：

1. 统一选择 trigger 或显式 SQL 更新策略。
2. 保证所有业务表行为一致。
3. 测试关键表更新后 `updated_at` 变化。

关联 GAP：

```text
GAP-2-008
```
