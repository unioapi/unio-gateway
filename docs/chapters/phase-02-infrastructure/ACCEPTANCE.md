# Phase 2 Acceptance

## 功能验收

1. config 能加载 HTTP、PostgreSQL、Redis 基础配置。
2. PostgreSQL pool 可以连接并执行 sqlc query。
3. Redis client 可以连接并执行基础命令。
4. migrations 能完整创建和回滚当前 schema。

## 生产验收

1. provider/channel/model/price 不作为正式 env/config 来源。
2. PostgreSQL pool 参数和健康检查可配置。
3. Redis timeout、pool size、namespace 可配置。
4. 启动前校验 schema version。
5. `updated_at` 策略统一。

## 测试验收

1. config 测试覆盖默认值和 env 覆盖。
2. sqlc 关键 query 测试通过。
3. Redis 限流相关测试不依赖真实生产 Redis。

## 文档验收

1. 所有基础设施 production TODO 都有 GAP 编号。
2. migration runner 和第三方库选择写入第三方库策略。

