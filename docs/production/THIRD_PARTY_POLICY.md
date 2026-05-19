# Third Party Policy

本文档用于判断一个能力应该自研，还是优先使用成熟第三方库。

## 基本原则

Unio API 不追求“所有东西都自己写”。

判断标准：

```text
业务核心必须可审计、可控、可解释。
通用基础设施应优先使用成熟库或标准工具。
安全和加密相关能力不要自造轮子。
```

## 应该优先自研的部分

这些部分是产品语义和商业核心：

| 能力 | 原因 |
| --- | --- |
| billing settlement | 直接决定用户扣费和平台收入，必须可审计。 |
| ledger 语义 | 账本是资金事实，必须精确控制事务和幂等。 |
| routing 策略 | 决定 provider/channel/model 的商业调度。 |
| price snapshot | 支撑历史账单复算和审计。 |
| request/attempt 状态机 | 决定失败、重试、取消、fallback 的业务事实。 |
| adapter contract | 是 Unio 内部边界，不能被 provider SDK 反向塑形。 |

## 应该优先评估第三方库的部分

| 能力 | 推荐方向 |
| --- | --- |
| migration runner | 使用成熟 migration 工具，避免手写 schema 版本管理。 |
| decimal / money | 使用 decimal 库或数据库 NUMERIC，避免 float。 |
| JWT | 使用成熟 JWT 库并明确算法白名单。 |
| argon2id | 使用 Go 官方 x/crypto 实现。 |
| Prometheus | 使用官方 client。 |
| OpenTelemetry | 使用官方 SDK。 |
| SSE parser | 优先评估成熟 parser；如果自研，必须有大 chunk、tool_calls、backpressure 测试。 |
| Redis rate limit atomicity | 使用 Lua 或成熟限流组件，不能长期依赖非原子 INCR + EXPIRE。 |
| UUID/ULID | 使用成熟库或标准随机实现，避免自造弱 ID。 |

## 引入依赖前检查

新增第三方库前必须回答：

1. 这个能力是不是 Unio 的业务核心？
2. 依赖是否维护活跃、许可证是否适合商业项目？
3. 是否会把第三方库类型泄漏到核心业务接口？
4. 是否能被测试替身替换？
5. 失败模式是否可控？
6. 是否会影响 adapter/gateway/routing/billing 的边界？

## 当前依赖

| 依赖 | 用途 | 判断 |
| --- | --- | --- |
| github.com/go-chi/chi/v5 | HTTP router | 合理。只留在 HTTP 层。 |
| github.com/jackc/pgx/v5 | PostgreSQL driver/pool | 合理。核心 SQL 继续保持显式。 |
| github.com/redis/go-redis/v9 | Redis client | 合理。后续需要补 timeout/pool/namespace 配置。 |

