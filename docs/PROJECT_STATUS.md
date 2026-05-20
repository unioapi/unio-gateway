# Project Status

更新时间：2026-05-20

实现主线：

```text
阶段 7：计费与账本
当前建议小节：7.17 余额预检查与预授权最小闭环
```

当前协作焦点：

```text
从项目起点开始逐章复盘，不直接推进阶段 7。
2026-05-20 本轮已从阶段 1 扫到阶段 5：
1. 阶段核心目标是否已经立住。
2. 哪些欠账适合现在补。
3. 哪些欠账应和后续阶段一起补。
4. 哪些欠账只需要在公开生产前关闭。
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

重要产品判断：

1. tool calling、function calling、streaming tool delta、structured output 和 multimodal input 都是商业 API 网关必须支持的能力。
2. 当前 text-only MVP 不假装支持这些能力；SSE parser 已补稳，后续进入正式实现前仍必须设计 tool/multimodal DTO、adapter contract、stream delta、usage/billing 和 fallback 语义。
3. 下次继续逐章复盘时，进入阶段 6，判断模型、渠道、routing、credential 和 project visibility 欠账哪些需要现在处理。

验证状态：

```bash
go test ./...
```

最近一次验证通过：2026-05-20。

## 阶段总览

| 阶段 | 名称 | 状态 | 当前判断 |
| --- | --- | --- | --- |
| 阶段 1 | [Go Web 骨架](chapters/phase-01-go-web/STATUS.md) | partial | 基础骨架已完成，HTTP timeout 和 request id 输入约束已收口；startup timeout/readiness 仍是生产欠账。 |
| 阶段 2 | [基础设施](chapters/phase-02-infrastructure/STATUS.md) | partial | PostgreSQL、Redis、migration、sqlc 基础能力已完成；PostgreSQL pool 和 Redis timeout/pool/retry/namespace 已生产化，migration runner 和 schema 版本检查未生产化。 |
| 阶段 3 | [用户与 API Key](chapters/phase-03-identity-api-key/STATUS.md) | partial | 用户、project、API key、认证、基础限流已完成，默认限流配置、Redis namespace、Redis 原子计数和故障策略已收口；API key 创建已校验 actor/project 归属，list、revoke、disable 和 audit log 仍未完成。 |
| 阶段 4 | [OpenAI-compatible API](chapters/phase-04-openai-compatible-api/STATUS.md) | partial | `/v1/models`、`/v1/chat/completions`、SSE 基础入口、严格 JSON 和 Chat DTO text-only 校验已完成；project 模型可见性和 SSE 写出后观测随阶段 6/7/8 收口。 |
| 阶段 5 | [Adapter 边界](chapters/phase-05-adapter-boundary/STATUS.md) | partial | adapter 接口、OpenAI 非流式/流式、usage 映射、当前 HTTP DTO 可透传参数 contract 和项目级 SSE event reader 已完成；provider error metadata 进入阶段 8 观测主线。 |
| 阶段 6 | [模型与渠道](chapters/phase-06-model-channel-routing/STATUS.md) | partial | provider/channel/model/routing/fallback 基础完成，project 可见性、credential 正式解析和装配治理未完成。 |
| 阶段 7 | [计费与账本](chapters/phase-07-billing-ledger/STATUS.md) | in_progress | request/attempt/usage/ledger/settlement 和 stream final usage 已完成，pre-authorize、状态机、幂等和成本快照是当前 P0/P1。 |
| 阶段 8 | [可观测性与稳定性](chapters/phase-08-observability-stability/STATUS.md) | planned | 尚未正式进入。当前只有少量 adapter metadata 相关前置 TODO。 |
| 阶段 9 | [后台管理](chapters/phase-09-admin/STATUS.md) | planned | 尚未正式进入。进入前必须先处理 credential resolver 和后台管理边界。 |

## 当前上线阻断

当前不应进入生产公开计费 API，原因：

1. 非流式请求没有余额预检或预授权。
2. 流式请求没有预授权、capture、refund 闭环。
3. settlement 缺少请求级幂等完成检测。
4. request/attempt 终态更新缺少状态机守卫。
5. 无 final usage 的 stream 中断策略还不能覆盖平台成本控制。
6. stream 写出后错误观测仍依赖 request 状态和后续 observability 收口。

## 下一步

当前有两类下一步，按协作目标选择：

1. 本轮逐章复盘：下一步进入阶段 6 [模型与渠道](chapters/phase-06-model-channel-routing/STATUS.md)，复核 provider/channel/model/routing/credential 的当前生产欠账。
2. 回到实现主线时：进入 [7.17 余额预检查与预授权最小闭环](chapters/phase-07-billing-ledger/PLAN.md#task-7-17-preauthorization)。

阶段 7 下一小节目标：

1. 明确余额不足请求在调用上游前的拒绝策略。
2. 为非流式和流式统一设计 reservation/pre-authorization。
3. 成功后按真实 usage capture。
4. 失败、取消、无 final usage 时按策略 refund 或保留异常记录。
5. 保证 ledger-first 和幂等语义不被破坏。
