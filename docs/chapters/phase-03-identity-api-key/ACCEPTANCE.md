# Phase 3 Acceptance

## 功能验收

1. 能创建 API key。
2. 数据库只保存 key hash 和 prefix。
3. 认证 middleware 能识别有效 key 并拒绝无效 key。
4. request context 中包含 user、project、api key 身份。

## 生产验收

1. key 创建接口有调用者授权检查。
2. 支持 revoke、disable、list。
3. 后台 key 操作有审计日志。
4. Redis 限流原子性可靠。
5. Redis 故障策略明确且可配置。

## 测试验收

1. API key 生成、hash、验证测试通过。
2. auth middleware 测试覆盖成功、缺失、无效、禁用 key。
3. rate limit 测试覆盖通过、超限、Redis 故障策略。

## 文档验收

1. key 管理和限流生产欠账登记到 TODO register。
2. API key 作为 customer/project 身份入口的边界写入阶段状态。

