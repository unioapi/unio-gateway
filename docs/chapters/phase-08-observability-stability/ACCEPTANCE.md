# Phase 8 Acceptance

## 功能验收

1. structured logs 带 request/correlation/project/model/provider/channel 字段。
2. metrics 能按 project/model/provider/channel 聚合。
3. adapter 能提供 upstream metadata。
4. retry/fallback 基于结构化错误分类。

## 生产验收

1. 日志不泄漏 API key、credential、上游敏感错误。
2. metrics label 不产生高基数风险。
3. tracing 不记录敏感请求内容。
4. channel health 能影响 routing。

## 测试验收

1. provider error classification 测试覆盖 401/403/429/5xx/timeout/cancel。
2. retry/fallback 测试覆盖可重试和不可重试错误。
3. stream 已写出后不 fallback 的测试通过。

## 文档验收

1. 观测字段和脱敏规则写入章节文档。
2. retry/fallback 策略写入决策或阶段计划。

