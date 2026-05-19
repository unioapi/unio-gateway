# Phase 4 Acceptance

## 功能验收

1. `GET /v1/models` 返回 OpenAI-compatible list。
2. `POST /v1/chat/completions` 支持非流式。
3. `POST /v1/chat/completions` 支持 SSE stream。
4. 错误响应使用 OpenAI-compatible error structure。

## 生产验收

1. HTTP DTO 深度校验完整。
2. 不支持的参数不被静默忽略。
3. JSON decode 严格且错误稳定。
4. SSE 写出后的错误通过 request 状态和观测系统可追踪。

## 测试验收

1. models handler 测试通过。
2. chat handler 非流式和流式测试通过。
3. JSON decode 测试覆盖 malformed、too large、trailing token。
4. DTO validation 测试覆盖 role/content/参数边界。

## 文档验收

1. OpenAI-compatible API 边界不与 adapter wire DTO 混淆。
2. 所有 HTTP 层生产欠账有 GAP 编号。

