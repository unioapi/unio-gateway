# Phase 7 Acceptance

## 功能验收

1. request record 和 attempt record 覆盖非流式、流式、失败、取消。
2. usage record 记录 prompt、completion、total、cached、reasoning token。
3. price snapshot 支撑历史账单复算。
4. ledger entry 是余额变化事实来源。
5. 非流式成功请求能 settlement。
6. stream final usage 能被解析并 settlement。

## 生产验收

1. 调用上游前有余额 preflight 或 reservation。
2. stream 有 freeze/capture/release 闭环。
3. 无 final usage、客户端取消、上游中断都有明确计费策略。
4. settlement 以 request 为边界幂等。
5. request/attempt 终态不会被覆盖。
6. 错误详情脱敏并区分用户可见和内部诊断。
7. 成本价和售卖价能分离审计。
8. 价格生效窗口不会重叠导致结算不确定。
9. 低余额但 `available_balance > 0` 的请求支持部分余额授权：冻结全部可用余额，允许请求继续。
10. 上游成功且有可靠 usage 时，`actual_amount > authorized_amount` 必须 capture 已冻结金额、记录平台差额核销，并让 request 成功收口；不得让用户余额变负或产生隐性欠费。

## 测试验收

1. billing 价格计算测试通过。
2. ledger credit/debit 事务测试通过。
3. settlement 成功、失败、重复执行测试通过。
4. stream final usage 成功结算测试通过。
5. stream 客户端取消、有 usage、无 usage 测试通过。
6. authorization baseline 的冻结、capture、release、余额不足测试通过。
7. 部分余额授权和平台差额核销测试通过，包括非流式、流式、actual 小于冻结、actual 等于冻结、actual 大于冻结。

## 文档验收

1. 所有阶段 7 production TODO 都有 GAP 编号。
2. TODO register 中每个阶段 7 GAP 都链接到具体 TASK。
3. 阶段 7 完成前，`docs/production/RELEASE_BLOCKERS.md` 中不得保留阶段 7 P0 blocker。
