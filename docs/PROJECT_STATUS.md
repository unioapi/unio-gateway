# Project Status

更新时间：2026-05-19

当前主线：

```text
阶段 7：计费与账本
当前建议小节：7.17 余额预检查与预授权最小闭环
```

## 阶段总览

| 阶段 | 名称 | 状态 | 当前判断 |
| --- | --- | --- | --- |
| 阶段 1 | Go Web 骨架 | partial | 基础骨架已完成，公网级 timeout/readiness/request id 输入约束仍是生产欠账。 |
| 阶段 2 | 基础设施 | partial | PostgreSQL、Redis、migration、sqlc 基础能力已完成，migration runner、pool 参数和 schema 版本检查未生产化。 |
| 阶段 3 | 用户与 API Key | partial | 用户、project、API key、认证、基础限流已完成，key 管理授权审计和限流降级策略未完成。 |
| 阶段 4 | OpenAI-compatible API | partial | `/v1/models`、`/v1/chat/completions`、SSE 基础入口已完成，DTO 深度校验和严格 JSON 边界未完成。 |
| 阶段 5 | Adapter 边界 | partial | adapter 接口、OpenAI 非流式/流式、usage 映射已完成，完整参数透传和更稳健 SSE parser 未完成。 |
| 阶段 6 | 模型与渠道 | partial | provider/channel/model/routing/fallback 基础完成，project 可见性、credential 正式解析和装配治理未完成。 |
| 阶段 7 | 计费与账本 | in_progress | request/attempt/usage/ledger/settlement 和 stream final usage 已完成，pre-authorize、状态机、幂等和成本快照是当前 P0/P1。 |
| 阶段 8 | 可观测性与稳定性 | planned | 尚未正式进入。当前只有少量 adapter metadata 相关前置 TODO。 |
| 阶段 9 | 后台管理 | planned | 尚未正式进入。进入前必须先处理 credential resolver 和后台管理边界。 |

## 当前上线阻断

当前不应进入生产公开计费 API，原因：

1. 非流式请求没有余额预检或预授权。
2. 流式请求没有预授权、capture、refund 闭环。
3. settlement 缺少请求级幂等完成检测。
4. request/attempt 终态更新缺少状态机守卫。
5. 无 final usage 的 stream 中断策略还不能覆盖平台成本控制。
6. chat request 深度校验不足，公开 API 可能接受非法参数。
7. 部分 OpenAI-compatible 参数已被 HTTP DTO 接收，但尚未进入 adapter contract，存在静默丢参风险。

## 下一步

优先进入：

```text
7.17 余额预检查与预授权最小闭环
```

本小节目标：

1. 明确余额不足请求在调用上游前的拒绝策略。
2. 为非流式和流式统一设计 reservation/pre-authorization。
3. 成功后按真实 usage capture。
4. 失败、取消、无 final usage 时按策略 refund 或保留异常记录。
5. 保证 ledger-first 和幂等语义不被破坏。
