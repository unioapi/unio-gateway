# Phase 7 Acceptance

## 功能验收

1. request record 和 attempt record 覆盖非流式、流式、失败、取消。
2. usage record 记录 prompt、completion、total、cached、reasoning token。
3. price snapshot 支撑历史账单复算。
4. ledger entry 是余额变化事实来源。
5. 非流式成功请求能 settlement。
6. stream final usage 能被解析并 settlement。

## 生产验收

1. 调用上游前有余额 preflight 或 pre-authorization。
2. stream 有 freeze/capture/release 闭环。
3. 无 final usage、客户端取消、上游中断都有明确计费策略。
4. settlement 以 request 为边界幂等。
5. request/attempt 终态不会被覆盖。
6. 错误详情脱敏并区分用户可见和内部诊断。
7. 成本价和售卖价能分离审计。
8. 价格生效窗口不会重叠导致结算不确定。

## 测试验收

1. billing 价格计算测试通过。
2. ledger credit/debit 事务测试通过。
3. settlement 成功、失败、重复执行测试通过。
4. stream final usage 成功结算测试通过。
5. stream 客户端取消、有 usage、无 usage 测试通过。
6. preauthorization 实现后补冻结、capture、release、余额不足测试。

## 文档验收

1. 所有阶段 7 production TODO 都有 GAP 编号。
2. TODO register 中每个阶段 7 GAP 都链接到具体 TASK。
3. 阶段 7 完成前，`docs/production/RELEASE_BLOCKERS.md` 中不得保留阶段 7 P0 blocker。
