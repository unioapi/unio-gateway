# TODO Register

本文档是生产欠账的事实登记表。

规则：

1. 每个生产 TODO 必须有稳定 `GAP-*` 编号。
2. 每个 GAP 必须链接到对应章节 `PLAN.md` 的具体任务。
3. 代码里的 TODO 只放简短上下文，完整风险和计划以本文档为准。
4. 关闭 GAP 时必须同步更新代码 TODO、章节 STATUS 和本文档状态。

## 状态定义

| 状态 | 含义 |
| --- | --- |
| todo | 已识别，尚未开始。 |
| in_progress | 正在处理。 |
| done | 已完成，代码 TODO 应移除。 |
| deferred | 已确认后移到后续阶段。 |

## 优先级定义

| 优先级 | 含义 |
| --- | --- |
| P0 | 上线阻断，涉及资金、安全、账务事实或公开 API 契约。 |
| P1 | 生产重要风险，应在对应阶段收口前完成。 |
| P2 | 生产增强或稳定性改进，可排期但不能遗忘。 |

## GAP 列表

| ID | 阶段 | 优先级 | 状态 | 上线阻断 | 代码位置 | 章节任务 | 风险摘要 | 计划处理时机 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| GAP-1-001 | 阶段 1 | P2 | todo | no | [cmd/server/main.go](../../cmd/server/main.go) | [TASK-1.02](../chapters/phase-01-go-web/PLAN.md#task-1-02-server-timeouts-readiness) | HTTP server timeout、shutdown timeout 和 readiness 未配置化。 | 公网部署前 |
| GAP-1-002 | 阶段 1 | P2 | todo | no | [internal/middleware/request_id.go](../../internal/middleware/request_id.go) | [TASK-1.03](../chapters/phase-01-go-web/PLAN.md#task-1-03-correlation-id) | 客户端 `X-Request-ID` 未限制，可能污染响应头和日志。 | 公网 API 前 |
| GAP-2-001 | 阶段 2 | P1 | todo | no | [cmd/server/main.go](../../cmd/server/main.go) | [TASK-2.04](../chapters/phase-02-infrastructure/PLAN.md#task-2-04-migration-runner) | 启动前没有 migration runner 或 schema version 检查。 | 生产部署前 |
| GAP-2-002 | 阶段 2 | P2 | todo | no | [migrations/000001_create_schema_health_checks.up.sql](../../migrations/000001_create_schema_health_checks.up.sql) | [TASK-2.04](../chapters/phase-02-infrastructure/PLAN.md#task-2-04-migration-runner) | 开发期 schema health table 的长期定位未确定。 | 引入正式 migration runner 时 |
| GAP-2-003 | 阶段 2 | P1 | todo | no | [internal/config/config.go](../../internal/config/config.go) | [TASK-2.02](../chapters/phase-02-infrastructure/PLAN.md#task-2-02-postgres-pool) | PostgreSQL pool 参数未纳入 config。 | 生产部署前 |
| GAP-2-004 | 阶段 2 | P1 | todo | no | [internal/config/config.go](../../internal/config/config.go) | [TASK-2.03](../chapters/phase-02-infrastructure/PLAN.md#task-2-03-redis-pool-namespace) | Redis timeout、pool、namespace 和降级策略未纳入 config。 | 生产部署前 |
| GAP-2-005 | 阶段 2 | P1 | todo | no | [internal/store/postgres.go](../../internal/store/postgres.go) | [TASK-2.02](../chapters/phase-02-infrastructure/PLAN.md#task-2-02-postgres-pool) | pgxpool 使用默认参数，生产连接池不可控。 | 生产部署前 |
| GAP-2-006 | 阶段 2 | P1 | todo | no | [internal/store/postgres.go](../../internal/store/postgres.go) | [TASK-2.04](../chapters/phase-02-infrastructure/PLAN.md#task-2-04-migration-runner) | 启动期未校验 migration 版本。 | 生产部署前 |
| GAP-2-007 | 阶段 2 | P2 | todo | no | [internal/redis/client.go](../../internal/redis/client.go) | [TASK-2.03](../chapters/phase-02-infrastructure/PLAN.md#task-2-03-redis-pool-namespace) | Redis client 使用默认连接参数。 | 生产部署前 |
| GAP-2-008 | 阶段 2 | P2 | todo | no | [migrations/000002_create_identity_tables.up.sql](../../migrations/000002_create_identity_tables.up.sql) | [TASK-2.05](../chapters/phase-02-infrastructure/PLAN.md#task-2-05-updated-at-strategy) | `updated_at` 更新策略未统一。 | 表更新逻辑扩展前 |
| GAP-3-001 | 阶段 3 | P2 | todo | no | [internal/auth/apikey.go](../../internal/auth/apikey.go) | [TASK-3.02](../chapters/phase-03-identity-api-key/PLAN.md#task-3-02-api-key-auth) | 每次认证同步更新 `last_used_at` 会放大数据库写入。 | 高流量前 |
| GAP-3-002 | 阶段 3 | P1 | todo | no | [internal/auth/apikey.go](../../internal/auth/apikey.go) | [TASK-3.03](../chapters/phase-03-identity-api-key/PLAN.md#task-3-03-api-key-management) | 缺少 API key revoke、disable、list 和审计日志。 | 开放 key 管理 API 前 |
| GAP-3-003 | 阶段 3 | P2 | todo | no | [cmd/server/main.go](../../cmd/server/main.go) | [TASK-3.04](../chapters/phase-03-identity-api-key/PLAN.md#task-3-04-rate-limit-production) | 默认 rate limit 阈值和窗口仍在启动装配中硬编码。 | 生产部署前 |
| GAP-3-004 | 阶段 3 | P1 | todo | no | [internal/ratelimit/redis_store.go](../../internal/ratelimit/redis_store.go) | [TASK-3.04](../chapters/phase-03-identity-api-key/PLAN.md#task-3-04-rate-limit-production) | Redis `INCR + EXPIRE` 非原子，可能留下无过期 key。 | 生产限流前 |
| GAP-3-005 | 阶段 3 | P2 | todo | no | [internal/ratelimit/redis_store.go](../../internal/ratelimit/redis_store.go) | [TASK-3.04](../chapters/phase-03-identity-api-key/PLAN.md#task-3-04-rate-limit-production) | Redis key namespace 未集中管理，环境隔离不足。 | 生产部署前 |
| GAP-3-006 | 阶段 3 | P2 | todo | no | [internal/middleware/rate_limit.go](../../internal/middleware/rate_limit.go) | [TASK-3.04](../chapters/phase-03-identity-api-key/PLAN.md#task-3-04-rate-limit-production) | Redis 限流故障会导致请求全部失败，策略不可配置。 | 生产部署前 |
| GAP-3-007 | 阶段 3 | P1 | todo | no | [internal/apikey/service.go](../../internal/apikey/service.go) | [TASK-3.03](../chapters/phase-03-identity-api-key/PLAN.md#task-3-03-api-key-management) | API key 创建缺少调用者授权和审计。 | 开放 key 管理 API 前 |
| GAP-4-001 | 阶段 4 | P0 | todo | yes | [internal/httpapi/chat_completions_handler.go](../../internal/httpapi/chat_completions_handler.go) | [TASK-4.03](../chapters/phase-04-openai-compatible-api/PLAN.md#task-4-03-chat-dto-validation) | chat message、role、content、stop/user 等 DTO 边界校验不足。 | 公开 OpenAI-compatible API 前 |
| GAP-4-002 | 阶段 4 | P1 | todo | no | [internal/httpx/json.go](../../internal/httpx/json.go) | [TASK-4.04](../chapters/phase-04-openai-compatible-api/PLAN.md#task-4-04-strict-json-error) | JSON decode 未校验 Content-Type 和尾随 token。 | 公开 OpenAI-compatible API 前 |
| GAP-5-001 | 阶段 5 | P0 | todo | yes | [internal/adapter/chat.go](../../internal/adapter/chat.go) | [TASK-5.01](../chapters/phase-05-adapter-boundary/PLAN.md#task-5-01-chat-parameter-contract) | HTTP DTO 接收的部分参数未进入 adapter contract，存在静默丢参。 | 公开 OpenAI-compatible chat API 前 |
| GAP-5-002 | 阶段 5 | P1 | todo | no | [internal/adapter/openai/chat.go](../../internal/adapter/openai/chat.go) | [TASK-5.03](../chapters/phase-05-adapter-boundary/PLAN.md#task-5-03-openai-stream-adapter) | `bufio.Scanner` 受单个 SSE event 大小上限影响。 | 支持 tool_calls 或大 chunk 前 |
| GAP-6-001 | 阶段 6 | P1 | deferred | no | [internal/credential/resolver.go](../../internal/credential/resolver.go) | [TASK-6.02](../chapters/phase-06-model-channel-routing/PLAN.md#task-6-02-credential-resolver) | 静态 credential resolver 无法支持安全轮换和后台管理。 | 阶段 9 前置 |
| GAP-6-002 | 阶段 6 | P2 | todo | no | [cmd/server/main.go](../../cmd/server/main.go) | [TASK-6.03](../chapters/phase-06-model-channel-routing/PLAN.md#task-6-03-bootstrap-wiring) | main 函数装配逻辑膨胀。 | 阶段 6 收口或后台装配前 |
| GAP-6-003 | 阶段 6 | P1 | todo | no | [cmd/server/main.go](../../cmd/server/main.go) | [TASK-6.03](../chapters/phase-06-model-channel-routing/PLAN.md#task-6-03-bootstrap-wiring) | provider.adapter 缺少启动/后台写入校验，可能运行时 registry miss。 | 启用真实 channel 或后台管理前 |
| GAP-6-004 | 阶段 6 | P1 | todo | no | [internal/config/config.go](../../internal/config/config.go) | [TASK-2.01](../chapters/phase-02-infrastructure/PLAN.md#task-2-01-config-boundary) | provider/channel 业务数据进入 config 会阻断后台动态管理。 | 接入数据库 channel 时 |
| GAP-6-005 | 阶段 6 | P1 | todo | no | [internal/routing/router.go](../../internal/routing/router.go) | [TASK-6.04](../chapters/phase-06-model-channel-routing/PLAN.md#task-6-04-routing-policy) | routing 未表达 project 级模型可见性、预算、禁用或专属 channel 策略。 | 多项目客户配置前 |
| GAP-6-006 | 阶段 6 | P1 | todo | no | [internal/modelcatalog/catalog.go](../../internal/modelcatalog/catalog.go) | [TASK-6.04](../chapters/phase-06-model-channel-routing/PLAN.md#task-6-04-routing-policy) | `/v1/models` 未体现 project 级可见性、预算或禁用策略。 | 后台项目配置前 |
| GAP-6-007 | 阶段 6 | P2 | todo | no | [internal/routing/router.go](../../internal/routing/router.go) | [TASK-6.05](../chapters/phase-06-model-channel-routing/PLAN.md#task-6-05-routing-error-semantics) | routing 无法区分模型不存在和无可用 channel。 | gateway 错误映射前 |
| GAP-7-001 | 阶段 7 | P0 | todo | yes | [internal/gateway/chat_completion.go](../../internal/gateway/chat_completion.go) | [TASK-7.17](../chapters/phase-07-billing-ledger/PLAN.md#task-7-17-preauthorization) | 非流式调用上游前没有余额预检或预授权。 | 公开计费 API 前 |
| GAP-7-002 | 阶段 7 | P0 | todo | yes | [internal/gateway/chat_stream.go](../../internal/gateway/chat_stream.go) | [TASK-7.17](../chapters/phase-07-billing-ledger/PLAN.md#task-7-17-preauthorization) | 流式请求调用上游前没有预授权，长输出或恶意断开无法控损。 | 公开 stream 计费 API 前 |
| GAP-7-003 | 阶段 7 | P0 | todo | yes | [sql/queries/request_records.sql](../../sql/queries/request_records.sql) | [TASK-7.18](../chapters/phase-07-billing-ledger/PLAN.md#task-7-18-request-state-machine) | request/attempt 状态更新没有状态机守卫。 | 补偿任务或并发 settlement 前 |
| GAP-7-004 | 阶段 7 | P0 | todo | yes | [internal/gateway/chat_request_record.go](../../internal/gateway/chat_request_record.go) | [TASK-7.17](../chapters/phase-07-billing-ledger/PLAN.md#task-7-17-preauthorization) | 无 final usage 的客户端取消缺少预授权，平台成本无法准确结算。 | 公开生产前 |
| GAP-7-005 | 阶段 7 | P1 | todo | no | [internal/gateway/chat_request_record.go](../../internal/gateway/chat_request_record.go) | [TASK-7.21](../chapters/phase-07-billing-ledger/PLAN.md#task-7-21-error-usage-audit) | request_records.error_message 保存原始内部错误，后台展示可能泄漏敏感细节。 | 开放请求日志查询前 |
| GAP-7-006 | 阶段 7 | P1 | todo | no | [internal/httpapi/chat_completions_handler.go](../../internal/httpapi/chat_completions_handler.go) | [TASK-4.05](../chapters/phase-04-openai-compatible-api/PLAN.md#task-4-05-sse-write-error) | SSE 写出后无法再返回 JSON error，客户端只能看到中断 stream。 | 公开生产前 |
| GAP-7-007 | 阶段 7 | P0 | todo | yes | [internal/gateway/chat_settlement.go](../../internal/gateway/chat_settlement.go) | [TASK-7.19](../chapters/phase-07-billing-ledger/PLAN.md#task-7-19-settlement-idempotency) | settlement 缺少请求级幂等完成检测。 | 补偿任务或并发重试前 |
| GAP-7-008 | 阶段 7 | P2 | todo | no | [internal/gateway/chat_settlement.go](../../internal/gateway/chat_settlement.go) | [TASK-7.21](../chapters/phase-07-billing-ledger/PLAN.md#task-7-21-error-usage-audit) | usage_records.source 无法区分非流式 response 和 stream final usage。 | stream billing 报表前 |
| GAP-7-009 | 阶段 7 | P1 | todo | no | [migrations/000006_create_price_tables.up.sql](../../migrations/000006_create_price_tables.up.sql) | [TASK-7.20](../chapters/phase-07-billing-ledger/PLAN.md#task-7-20-cost-snapshot) | prices 只表达客户侧售卖价，缺少成本价和 cost snapshot。 | 成本报表或多 channel 商业化前 |
| GAP-7-010 | 阶段 7 | P1 | todo | no | [migrations/000006_create_price_tables.up.sql](../../migrations/000006_create_price_tables.up.sql) | [TASK-7.22](../chapters/phase-07-billing-ledger/PLAN.md#task-7-22-price-effective-window) | 价格 enabled 生效窗口可能重叠，导致结算价格不确定。 | 开放价格后台管理前 |
| GAP-7-011 | 阶段 7 | P0 | todo | yes | [internal/ledger/service.go](../../internal/ledger/service.go) | [TASK-7.17](../chapters/phase-07-billing-ledger/PLAN.md#task-7-17-preauthorization) | ledger 缺少 pre-authorize、capture、refund 的冻结/释放语义。 | 公开计费 API 前 |
| GAP-7-012 | 阶段 7 | P0 | todo | yes | [internal/ledger/service.go](../../internal/ledger/service.go) | [TASK-7.19](../chapters/phase-07-billing-ledger/PLAN.md#task-7-19-settlement-idempotency) | 外部事务内并发 debit 幂等冲突会使事务失败且难以重入。 | 并发 settlement/补偿任务前 |
| GAP-8-001 | 阶段 8 | P1 | deferred | no | [internal/gateway/chat_settlement.go](../../internal/gateway/chat_settlement.go) | [TASK-8.01](../chapters/phase-08-observability-stability/PLAN.md#task-8-01-adapter-metadata-provider-errors) | adapter 成功响应缺少真实 upstream status/request id，影响渠道审计。 | 阶段 8 provider error classification 时 |

