# Project Status

更新时间：2026-05-26

实现主线：

```text
阶段 7：计费与账本
当前建议小节：7.20 成本价与毛利审计
```

当前协作焦点：

```text
阶段 7 billing 拆分、ledger reservation、冻结金额估算和 gateway authorization baseline 已完成。
TASK-7.21 已完成 safe/internal error 和 usage source 审计；GAP-7-005、GAP-7-008 已关闭。GAP-7-007 仍保留为 worker recovery 阻断项。
```

说明：

```text
实现主线停在阶段 7，不代表今天必须继续写阶段 7。
阶段 1-6 的 partial 表示仍有生产欠账，不等于这些阶段全部不可用。
Release blockers 表示公开生产前必须关闭，不等于每次学习或复盘都要先处理。
```

## 本轮交接摘要

已收口：

1. 阶段 1：HTTP timeout 和 request id 输入约束已收口；`GAP-1-002` 已关闭。
2. 阶段 2：PostgreSQL pool、Redis timeout/pool/retry/namespace 已配置化；`GAP-2-003`、`GAP-2-004`、`GAP-2-005`、`GAP-2-007` 已关闭。
3. 阶段 3：默认限流配置、Redis 原子计数、限流故障策略、API key 创建 actor/project 授权已收口；`GAP-3-003`、`GAP-3-004`、`GAP-3-005`、`GAP-3-006` 已关闭，`GAP-3-007` 已收窄为 audit log。
4. 阶段 4：严格 JSON decode 和 text-only Chat DTO 深度校验已完成；`GAP-4-001`、`GAP-4-002` 已关闭。
5. 阶段 5：当前 HTTP DTO 可透传参数已进入 `adapter.ChatRequest`、OpenAI wire DTO、非流式和流式请求；`GAP-5-001` 已关闭并移出 release blockers。
6. 阶段 5：OpenAI stream parser 已从逐行 `bufio.Scanner` 替换为项目级 SSE event reader；`GAP-5-002` 已关闭。
7. 阶段 5 学习交接已写入 [phase-05-adapter-boundary/HANDOFF.md](chapters/phase-05-adapter-boundary/HANDOFF.md)，下一次可从 `internal/adapter/sse` 开始学习。
8. Failure 结构化错误基础已接入主要模块，并写入 [phase-08-observability-stability/HANDOFF.md](chapters/phase-08-observability-stability/HANDOFF.md)，后续 provider error classification、retry/fallback 和 observability 以此为基础继续推进。
9. 阶段 7 ledger reservation schema、`reserved_balance`、`PreAuthorize`、`Capture`、`Release` 已完成。
10. 阶段 7 billing 冻结金额估算 `EstimateAuthorizationAmount` 已完成。
11. 阶段 7 gateway request-level authorization 已接入非流式和流式调用链，普通失败路径会 release，可能产生上游成本但无 final usage 的 stream 路径会 exception release。
12. 阶段 7 部分余额授权和平台差额核销已完成：`estimated_amount` 与 `authorized_amount` 已拆分，低余额可冻结全部可用余额并继续请求，`actual_amount > authorized_amount` 时写入 `ledger_billing_exceptions` 的 `write_off` 平台核销事实。
13. 阶段 7 无 final usage 的 stream 风险敞口已落地：客户端取消、emit 后中断和正常结束但缺 final usage 会释放用户冻结余额，并写入 `ledger_billing_exceptions` 的 `risk_exposure` 事实。
14. 阶段 7 provider/model 输入 token 估算已接入：gateway authorization 通过 adapter registry 调用 `ChatInputTokenizer`，OpenAI adapter 使用 `tiktoken-go/tokenizer` 按 upstream model 估算输入 token；`GAP-7-013` 已关闭。
15. 阶段 7 request/attempt 状态机守卫已完成：`request_records` 和 `request_attempts` 终态不会被并发补偿或重复更新覆盖，重复终态更新会读回第一次终态事实；`GAP-7-003` 已关闭并移出 release blockers。
16. 阶段 7 settlement 成功重放检查已完成：重复 `SettleSuccessfulChat` 会锁定 request，已成功请求只校验既有 usage、price snapshot、reservation、ledger 和 write-off 事实，一致才幂等成功。
17. 阶段 7 外部事务内 debit 幂等重入已完成：`DebitWithQueries` 在扣余额前按 ledger entry `idempotency_key` 获取 transaction-level advisory lock；`GAP-7-012` 已关闭并移出 release blockers。

重要产品判断：

1. tool calling、function calling、streaming tool delta、structured output 和 multimodal input 都是商业 API 网关必须支持的能力。
2. 当前 text-only MVP 不假装支持这些能力；SSE parser 已补稳，后续进入正式实现前仍必须设计 tool/multimodal DTO、adapter contract、stream delta、usage/billing 和 fallback 语义。
3. 阶段 6 已收口；credential 正式解析和 provider/project 后台策略推迟到阶段 9，预算约束推迟到阶段 7。
4. 阶段 7 计费产品规则已定：部分余额授权 + 平台差额核销；不允许用户负余额、隐性欠费或充值后追扣旧账。
5. 上游成功且已有可靠 usage 后，settlement 失败不能简单 release 冻结余额；该问题暂时不实现 goroutine 补偿，后续进入 worker/settlement recovery 线时用持久化任务和幂等重试收口。

验证状态：

```bash
go test ./...
```

最近一次验证通过：2026-05-26。

## 阶段总览

| 阶段 | 名称 | 状态 | 当前判断 |
| --- | --- | --- | --- |
| 阶段 1 | [Go Web 骨架](chapters/phase-01-go-web/STATUS.md) | partial | 基础骨架已完成，HTTP timeout 和 request id 输入约束已收口；startup timeout/readiness 仍是生产欠账。 |
| 阶段 2 | [基础设施](chapters/phase-02-infrastructure/STATUS.md) | partial | PostgreSQL、Redis、migration、sqlc 基础能力已完成；config 边界、PostgreSQL pool 和 Redis timeout/pool/retry/namespace 已生产化，migration runner 和 schema 版本检查未生产化。 |
| 阶段 3 | [用户与 API Key](chapters/phase-03-identity-api-key/STATUS.md) | partial | 用户、project、API key、认证、基础限流已完成，默认限流配置、Redis namespace、Redis 原子计数和故障策略已收口；API key 创建已校验 actor/project 归属，list、revoke、disable 和 audit log 仍未完成。 |
| 阶段 4 | [OpenAI-compatible API](chapters/phase-04-openai-compatible-api/STATUS.md) | partial | `/v1/models`、`/v1/chat/completions`、SSE 基础入口、严格 JSON 和 Chat DTO text-only 校验已完成；project 模型可见性和 SSE 写出后观测随阶段 6/7/8 收口。 |
| 阶段 5 | [Adapter 边界](chapters/phase-05-adapter-boundary/STATUS.md) | partial | adapter 接口、OpenAI 非流式/流式、usage 映射、当前 HTTP DTO 可透传参数 contract 和项目级 SSE event reader 已完成；provider error metadata 进入阶段 8 观测主线。 |
| 阶段 6 | [模型与渠道](chapters/phase-06-model-channel-routing/STATUS.md) | done | provider/channel/model/routing/fallback、project 模型 allow-list/deny-list、adapter/routing/gateway/http/server app bootstrap 和启动期 provider.adapter preflight 已接入；credential 正式解析和 provider/project 后台策略推迟到阶段 9，预算约束推迟到阶段 7。 |
| 阶段 7 | [计费与账本](chapters/phase-07-billing-ledger/STATUS.md) | in_progress | request/attempt/usage/ledger/settlement、stream final usage、ledger reservation、billing 冻结金额估算、gateway authorization baseline、部分余额授权、平台差额核销、无 final usage 风险敞口记录、输入 token 估算、request/attempt 状态机守卫、settlement 成功重放检查、外部事务内 debit 幂等重入、usage source 审计和 safe/internal error 审计已完成；worker recovery、成本快照和价格窗口仍未完成。 |
| 阶段 8 | [可观测性与稳定性](chapters/phase-08-observability-stability/STATUS.md) | planned | 尚未正式进入。当前只有少量 adapter metadata 相关前置 TODO。 |
| 阶段 9 | [后台管理](chapters/phase-09-admin/STATUS.md) | planned | 尚未正式进入。进入前必须先处理 credential resolver 和后台管理边界。 |

## 当前上线阻断

当前不应进入生产公开计费 API，原因：

1. 上游成功后首次 settlement 失败仍可能导致冻结余额悬挂；该问题保留为 worker/recovery 阶段处理，不在当前状态机小节实现。
2. stream 写出后错误观测仍依赖 request 状态和后续 observability 收口。

## 下一步

下一步可继续阶段 7 后续小节。settlement recovery 暂不插队，等进入 worker/settlement recovery 线时处理。

阶段 7 下一小节目标：

1. 继续推进 [GAP-7-009](production/TODO_REGISTER.md#gap-7-009) 成本价和 cost snapshot；倍率只作为后续后台运营配置工具，账务核心先落明确金额和请求级快照。
2. 继续推进 [GAP-7-010](production/TODO_REGISTER.md#gap-7-010) 价格生效窗口约束。
3. 后续进入 worker/recovery 线时处理 GAP-7-007。
