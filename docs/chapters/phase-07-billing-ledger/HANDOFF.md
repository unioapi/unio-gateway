# Phase 7 Handoff - Billing Ledger

更新时间：2026-05-28

## 当前状态

阶段 7 尚未完成，不应进入阶段 8。

当前阶段主线已经完成：

1. request / attempt / usage / price snapshot / ledger 基础链路。
2. gateway 请求前余额授权、冻结、capture、release。
3. 部分余额授权与平台差额核销。
4. stream final usage settlement 与无 final usage risk exposure。
5. request / attempt 状态机守卫。
6. settlement 成功重放一致性校验。
7. 客户售价与 provider/channel 成本价分离。
8. request-level `cost_snapshots` 成本快照写入与幂等重放校验。
9. `prices` enabled 生效窗口重叠约束。
10. 全服务目录结构改造。

当前剩余 P0 阻断项：

```text
GAP-7-007：上游成功且有可靠 usage 后，如果首次 settlement 失败，需要 worker 持久化 recovery job + 幂等 settlement 重试收口。
```

## 本班次交接重点

本班次完成了 4 组工作。

### 1. 成本价与成本快照已接入 settlement

已完成内容：

1. `SettleSuccessfulChat` 按最终 `channel + model` 和 attempt time 查询 active `channel_cost_prices`。
2. 使用 `billing.CalculateProviderCost` 计算 provider 成本分项和总成本。
3. 在同一笔 settlement 事务里写入 `cost_snapshots`。
4. 重复 settlement 成功重放时，读取既有 `cost_snapshots` 并校验：
   - request / provider / channel / model / upstream_model。
   - currency / pricing_unit / formula_version。
   - input / output / cached / reasoning 单价。
   - input / output / cached / reasoning / total 成本金额。
5. `GAP-7-009` 已关闭。

关键文件：

```text
internal/service/gateway/chat_settlement.go
internal/service/gateway/chat_settlement_test.go
internal/core/billing/service.go
internal/core/billing/types.go
sql/queries/channel_cost_prices.sql
sql/queries/cost_snapshots.sql
migrations/000019_create_channel_cost_prices.up.sql
migrations/000020_create_cost_snapshots.up.sql
```

### 2. 价格 enabled 生效窗口约束已完成

已完成内容：

1. `prices` 使用 PostgreSQL `btree_gist` + exclusion constraint。
2. 禁止同一 `model_id + currency + pricing_unit` 下 enabled 价格窗口重叠。
3. 使用 `[)` 时间区间，允许相邻窗口无缝切换。
4. disabled 价格草稿允许重叠。
5. 不同 model / currency / pricing_unit 不互相阻塞。
6. `GAP-7-010` 已关闭。

关键文件：

```text
migrations/000012_create_prices.up.sql
internal/platform/store/sqlc/prices_test.go
```

注意：

```text
CREATE EXTENSION IF NOT EXISTS btree_gist;
```

这是 PostgreSQL extension 语法，不是普通 SQL 标准语法。它让 GiST exclusion constraint 可以比较 `uuid`、`text` 这类普通等值字段。

### 3. 全服务目录结构改造已完成

目标结构已落地：

```text
cmd/gateway-server
cmd/admin-server
cmd/console-server
cmd/worker-server
internal/bootstrap
internal/app/gatewayapi
internal/service/gateway
internal/core/*
internal/platform/*
```

当前真实代码迁移：

```text
cmd/server                       -> cmd/gateway-server
internal/httpapi                 -> internal/app/gatewayapi
internal/gateway                 -> internal/service/gateway
internal/billing                 -> internal/core/billing
internal/ledger                  -> internal/core/ledger
internal/requestlog              -> internal/core/requestlog
internal/routing                 -> internal/core/routing
internal/adapter                 -> internal/core/adapter
internal/apikey                  -> internal/core/apikey
internal/auth                    -> internal/core/auth
internal/channel                 -> internal/core/channel
internal/modelcatalog            -> internal/core/modelcatalog
internal/credential              -> internal/core/credential
internal/config                  -> internal/platform/config
internal/store                   -> internal/platform/store
internal/redis                   -> internal/platform/redis
internal/httpx                   -> internal/platform/httpx
internal/failure                 -> internal/platform/failure
internal/ratelimit               -> internal/platform/ratelimit
internal/middleware/request_id   -> internal/platform/httpmw/request_id
internal/middleware/logger       -> internal/platform/httpmw/logger
internal/middleware/recoverer    -> internal/platform/httpmw/recoverer
internal/middleware/api_key_auth -> internal/app/gatewayapi/middleware/api_key_auth
internal/middleware/rate_limit   -> internal/app/gatewayapi/middleware/rate_limit
```

`sqlc.yaml` 已改为：

```text
out: internal/platform/store/sqlc
```

进程入口目录：

```text
cmd/gateway-server/main.go
cmd/admin-server/.gitkeep
cmd/console-server/.gitkeep
cmd/worker-server/.gitkeep
```

`.gitkeep` 只用于让 Git 跟踪目标入口目录，不代表 Admin API、Console API、Worker 已实现。

架构文档：

```text
docs/architecture/PROJECT_STRUCTURE.md
```

### 4. 文档状态已同步

已同步文档：

```text
AGENTS.md
docs/README.md
docs/PROJECT_STATUS.md
docs/architecture/PROJECT_STRUCTURE.md
docs/chapters/phase-07-billing-ledger/PLAN.md
docs/chapters/phase-07-billing-ledger/STATUS.md
docs/production/TODO_REGISTER.md
docs/production/PHASE7_PRODUCTION_TODO_AUDIT.md
```

`AGENTS.md` 中 API 前缀已固定为：

```text
/v1/*
/admin/v1/*
/console/v1/*
```

## 已完成 GAP

本班次关闭：

```text
GAP-7-009：provider/channel 成本价与 request-level cost snapshot。
GAP-7-010：价格 enabled 生效窗口重叠约束。
```

此前已关闭的阶段 7 关键 GAP：

```text
GAP-7-003：request/attempt 状态机守卫。
GAP-7-004：无 final usage 的 stream risk exposure。
GAP-7-005：safe/internal error 审计。
GAP-7-008：usage source 审计。
GAP-7-012：外部事务内 debit 幂等重入。
GAP-7-013：provider/model 输入 token 估算。
GAP-7-014：部分余额授权与平台差额核销。
```

## 仍需收口

下一步只剩阶段 7 worker recovery 主线。

必须先处理：

```text
GAP-7-007
```

业务语义：

```text
上游已经成功并返回可靠 usage 后，如果 SettleSuccessfulChat 失败，不能简单 release 冻结余额。
因为 provider 侧可能已经产生成本，业务语义应优先补偿重试 settlement/capture。
```

当前风险：

```text
request 可能被标记 failed。
reservation 可能保持 authorized。
reserved_balance 可能悬挂。
```

目标方案：

```text
worker 持久化 recovery job + 幂等 settlement 重试。
```

建议落点：

```text
cmd/worker-server/main.go
internal/bootstrap/worker_server.go
internal/app/workers/runner.go
internal/app/workers/settlement_recovery_worker.go
internal/service/gateway/chat_settlement_recovery.go
migrations/000021_create_settlement_recovery_jobs.up.sql
migrations/000021_create_settlement_recovery_jobs.down.sql
sql/queries/settlement_recovery_jobs.sql
```

建议顺序：

1. 设计 `settlement_recovery_jobs` 表。
2. 写 sqlc query：创建 job、claim job、mark succeeded、mark retry、mark failed。
3. 在 gateway 首次 settlement 失败路径创建 recovery job。
4. 在 `internal/service/gateway` 增加 recovery service，复用现有幂等 settlement 逻辑。
5. 在 `internal/app/workers` 增加 runner 和 settlement recovery worker。
6. 在 `cmd/worker-server` 增加真实 `main.go`，替换 `.gitkeep`。
7. 增加测试覆盖 job claim、幂等重试、成功收口、失败重试次数。

## 当前关键文件

结构入口：

```text
cmd/gateway-server/main.go
cmd/admin-server/.gitkeep
cmd/console-server/.gitkeep
cmd/worker-server/.gitkeep
internal/bootstrap/gateway_server.go
internal/bootstrap/http.go
```

Gateway API：

```text
internal/app/gatewayapi/router.go
internal/app/gatewayapi/chat_completions_handler.go
internal/app/gatewayapi/models_handler.go
internal/app/gatewayapi/middleware/api_key_auth.go
internal/app/gatewayapi/middleware/rate_limit.go
```

Gateway service：

```text
internal/service/gateway/chat_authorization.go
internal/service/gateway/chat_completion.go
internal/service/gateway/chat_stream.go
internal/service/gateway/chat_settlement.go
internal/service/gateway/service.go
```

Core：

```text
internal/core/billing
internal/core/ledger
internal/core/requestlog
internal/core/routing
internal/core/adapter
internal/core/apikey
internal/core/auth
internal/core/channel
internal/core/modelcatalog
internal/core/credential
```

Platform：

```text
internal/platform/config
internal/platform/store
internal/platform/store/sqlc
internal/platform/redis
internal/platform/httpx
internal/platform/httpmw
internal/platform/ratelimit
internal/platform/failure
```

文档：

```text
AGENTS.md
docs/README.md
docs/PROJECT_STATUS.md
docs/architecture/PROJECT_STRUCTURE.md
docs/chapters/phase-07-billing-ledger/PLAN.md
docs/chapters/phase-07-billing-ledger/STATUS.md
docs/production/TODO_REGISTER.md
docs/production/RELEASE_BLOCKERS.md
docs/production/DECISIONS.md
```

## 注意事项

1. 不要退回“用户必须自己算 token 才能调用”的产品体验。
2. 不要实现隐性欠费、负余额或充值后追扣；如果未来要做信用额度，必须另开决策和账务模型。
3. `estimated_amount` 和 `authorized_amount` 是两个概念，不能重新混用。
4. 上游成功且有可靠 usage 时，`actual_amount > authorized_amount` 不应导致普通 settlement failed。
5. write-off 必须是可审计账务事实，不能只写日志。
6. stream 不能只复用非流式后扣费模型，需要 authorization、release、capture、write-off 和 risk-exposure 语义。
7. ledger-first 不能被绕过，余额变化和核销都必须有账务事实。
8. 所有补偿和重试都要考虑幂等。
9. 上游成功且有可靠 usage 后 settlement 失败，不要直接 release；必须由 worker recovery 收口。
10. 成本价不要做倍率，不要只按 provider 定价，第一版按 channel + model 维护明确金额。
11. cost snapshot 不能只存单价，必须同时保存请求级实际平台成本金额。
12. 历史成本、毛利和审计只能依赖请求级快照事实，不能用当前成本价配置回算历史。
13. 新代码必须进入新目录结构，不要再使用迁移前路径。
14. `cmd/admin-server`、`cmd/console-server`、`cmd/worker-server` 当前只有 `.gitkeep`，不要误认为进程已实现。

## 最近验证

最近一次全量验证：

```bash
sqlc generate
go test ./...
go list ./...
git diff --check
```

结果：通过，时间为 2026-05-28。
