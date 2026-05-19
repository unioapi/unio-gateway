# Phase 5 Acceptance

## 功能验收

1. gateway 只依赖 adapter interface。
2. OpenAI 非流式请求可用。
3. OpenAI stream 请求可用。
4. OpenAI usage 映射包含 cached/reasoning tokens。

## 生产验收

1. 用户传入的 supported 参数不会静默丢失。
2. 不支持参数会被明确拒绝或记录为明确产品限制。
3. stream parser 能处理大 chunk、tool_calls、backpressure 和异常 SSE event。
4. adapter 不读取 provider/channel env，不查询数据库。

## 测试验收

1. adapter request/response 映射测试通过。
2. stream delta 和 final usage 测试通过。
3. cached/reasoning usage 测试通过。
4. 大 chunk parser 场景在替换 parser 后补齐。

## 文档验收

1. adapter DTO、HTTP DTO、OpenAI wire DTO 边界写清楚。
2. 所有 adapter 生产欠账有 GAP 编号。

