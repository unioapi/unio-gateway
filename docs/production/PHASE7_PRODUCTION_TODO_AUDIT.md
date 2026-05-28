# Phase 7 Production TODO Audit

## 后续治理补充

本文档是阶段 7 生产欠账审计报告。

审计之后，项目已引入统一文档治理结构：

```text
docs/production/TODO_REGISTER.md
```

所有 production TODO 已补充稳定 `GAP-*` 编号，并以 TODO register 为当前事实来源。本文档保留为阶段 7 审计快照，后续新增、关闭和改期生产欠账时，应优先维护 TODO register、章节 `PLAN.md` 和章节 `STATUS.md`。

2026-05-23 更新：

```text
GAP-7-001、GAP-7-002、GAP-7-011 已由 gateway authorization baseline 关闭。
余额策略后续按 GAP-7-014 收口：部分余额授权 + 平台差额核销。
本文后续条目仍保留原始审计快照，不作为当前状态事实来源。
```

## 背景

本报告记录一次针对 Unio API 阶段 1-7 已实现代码的生产欠账审计。

触发原因：

```text
第 7 阶段功能闭环已经跑通，但从商业项目视角看，仍存在若干生产级风险没有明确 TODO 标记。
```

审计目标：

```text
1. 扫描阶段 7 及之前已经进入实现边界的代码。
2. 以商业 API 网关的生产上线标准复盘欠账。
3. 只补充当前阶段或已完成阶段相关的 production TODO。
4. 避免把阶段 8/9 以后才应该处理的远期能力提前塞进代码。
5. 输出新增 TODO 的位置、风险、计划完善时机和建议优先级。
```

验证命令：

```text
go test ./...
go vet ./...
```

验证结果：

```text
全部通过
```

---

## 审计范围

本次审计覆盖：

```text
cmd/gateway-server
internal/platform/config
internal/app/gatewayapi
internal/platform/httpx
internal/app/gatewayapi/middleware 与 internal/platform/httpmw
internal/core/auth
internal/core/apikey
internal/platform/ratelimit
internal/core/adapter
internal/core/adapter/openai
internal/core/routing
internal/core/modelcatalog
internal/service/gateway
internal/core/billing
internal/core/ledger
internal/core/requestlog
internal/platform/store
migrations
sql/queries
```

重点检查：

```text
HTTP 输入边界
认证与 API Key 管理
限流降级
OpenAI-compatible 参数转发
stream parser 风险
model / routing project scope
price / cost / snapshot 边界
request / attempt 状态机
settlement 幂等
余额预检 / 预授权
stream 无 final usage 风险
错误信息脱敏
```

---

## 新增 TODO 总览

本次新增 production TODO 数量：

```text
19
```

按阶段分布：

```text
阶段 1: 1
阶段 3: 2
阶段 4: 2
阶段 5: 2
阶段 6: 2
阶段 7: 10
```

其中阶段 7 新增最多，原因是第 7 阶段已经进入商业计费主路径，生产风险不再只是工程质量问题，而是会直接影响：

```text
平台成本
用户余额
账单审计
退款/补偿
毛利统计
滥用风险
```

---

## 新增 TODO 明细

### 1. Request ID 输入约束

位置：

```text
internal/platform/httpmw/request_id.go:19
```

新增 TODO：

```go
// TODO(阶段1/production): [GAP-1-002] 直接信任客户端 X-Request-ID 会导致超长值或控制字符进入响应头和日志；开放公网 API 前；限制长度/字符集，非法时生成服务端 correlation id。
```

风险：

```text
客户端可传入任意 X-Request-ID。
如果值过长或带控制字符，可能污染响应头、日志和后续 tracing。
```

计划完善时机：

```text
开放公网 API 前。
```

建议优先级：

```text
P2
```

---

### 2. API Key 创建缺少审计日志

位置：

```text
internal/core/apikey/service.go:55
```

当前 TODO：

```go
// TODO(阶段3/production): [GAP-3-007] API Key 创建缺少审计日志；开放 key 管理 API 前；接入 audit log 记录 actor、project、api_key 和操作结果。
```

风险：

```text
API key 创建已传入 ActorUserID，并校验 actor/project 归属。
当前剩余风险是敏感操作没有审计记录，未来后台排查、合规和越权追责缺少事实来源。
```

计划完善时机：

```text
开放 API Key 管理 API 前。
```

建议优先级：

```text
P1
```

---

### 3. Redis 限流故障策略缺失

位置：

```text
internal/app/gatewayapi/middleware/rate_limit.go:39
```

关闭记录：

```text
GAP-3-006 已于 2026-05-20 关闭。Redis 限流故障策略已支持 fail_closed / fail_open 配置，并记录脱敏 structured log；Prometheus metrics 进入阶段 8 TASK-8.02 统一实现。
```

风险：

```text
Redis 故障会导致所有受限流保护的请求失败。
商业 API 需要明确限流故障时是 fail-open 还是 fail-closed。
```

实际处理时间：

```text
2026-05-20 已完成。
```

建议优先级：

```text
P2
```

---

### 4. JSON body 解码不够严格

位置：

```text
internal/platform/httpx/json.go:25
```

关闭记录：

```text
GAP-4-002 已于 2026-05-20 关闭。DecodeJSON 已校验 Content-Type、空 body、超大 body 和尾随 JSON token，chat completions handler 已映射为稳定 OpenAI-compatible error。
```

风险：

```text
已通过严格 JSON decode 收口。
后续如果新增其他 JSON endpoint，应复用 httpx.DecodeJSON，避免重新引入宽松解析。
```

实际处理时间：

```text
2026-05-20 已完成。
```

建议优先级：

```text
P2
```

---

### 5. Chat request 深度校验不足

位置：

```text
internal/app/gatewayapi/chat_completions_handler.go:55
```

关闭记录：

```text
GAP-4-001 已于 2026-05-20 关闭。Chat DTO 已校验 model、message role/content、temperature/top_p/max_tokens、presence/frequency penalty、stop 和 user 边界，并保持 OpenAI-compatible error 格式。
```

风险：

```text
text-only MVP 的 chat request 深度校验已收口。
tool/function/developer role 与 multimodal content 尚未进入 DTO 和 adapter contract，因此当前不会被假支持。
```

实际处理时间：

```text
2026-05-20 已完成。
```

建议优先级：

```text
P1
```

---

### 6. Adapter contract 未承载 OpenAI-compatible 可选参数

位置：

```text
internal/core/adapter/chat.go:20
```

关闭记录：

```text
GAP-5-001 已于 2026-05-20 关闭。adapter.ChatRequest、OpenAI wire DTO、非流式和流式上游请求均已承载 HTTP DTO 当前可透传参数，并有 gateway 与 OpenAI adapter 测试覆盖。
```

风险：

```text
静默丢参风险已通过 adapter contract 参数穿透收口。
role/content/参数值深度校验已由 GAP-4-001 收口。
```

实际处理时间：

```text
2026-05-20 已完成。
```

建议优先级：

```text
P1
```

---

### 7. OpenAI stream parser 使用 Scanner 存在长 event 风险

位置：

```text
internal/core/adapter/openai/chat.go:153
```

新增 TODO：

```go
// TODO(阶段5/production): [GAP-5-002] bufio.Scanner 仍受单个 SSE event 大小上限影响，遇到超长 delta/tool_calls 可能中断 stream；支持工具调用或大 chunk 上游前；改为基于 reader 的 SSE event parser，并显式处理 backpressure 和超限错误。
```

风险：

```text
Scanner 有 token size 限制。
当前虽然调高到 1MB，但未来 tool_calls、大 delta 或异常上游仍可能触发 scanner.Err。
```

计划完善时机：

```text
支持 tool calls 或大 chunk 上游前。
```

建议优先级：

```text
P2
```

---

### 8. /v1/models 缺少 project policy

位置：

```text
internal/core/modelcatalog/catalog.go:32
```

新增 TODO：

```go
// TODO(阶段6/production): [GAP-6-006] /v1/models 当前只按全局 enabled channel/model 返回，未体现 project 级可见性、预算或禁用策略；开放后台项目配置前；与 routing 共用 project model/channel policy，保证“可见模型”和“可路由模型”一致。
```

风险：

```text
用户看到的模型列表只取决于全局 enabled 状态。
未来 project 级权限、预算、模型禁用、专属渠道接入后，模型可见性会不准确。
```

计划完善时机：

```text
开放后台项目配置前。
```

建议优先级：

```text
P1
```

---

### 9. Routing 缺少 project model/channel policy

位置：

```text
internal/core/routing/router.go:96
```

新增 TODO：

```go
// TODO(阶段6/production): [GAP-6-005] routing 当前只校验 project_id 大于 0，尚未表达 project 级模型可见性、预算、禁用或专属 channel 策略；开放多项目客户配置前；引入 project_model/channel policy 查询并让 /v1/models 与 routing 共用同一可见性规则。
```

风险：

```text
routing 查询中 project_id 目前只是占位。
没有真正限制某个 project 能用哪些模型、哪些 channel。
```

计划完善时机：

```text
开放多项目客户配置前。
```

建议优先级：

```text
P1
```

---

### 10. 缺少 provider/channel 成本价快照

位置：

```text
migrations/000012_create_prices.up.sql:2
```

新增 TODO：

```sql
-- TODO(阶段7/production): [GAP-7-009] prices 当前只表达客户侧售卖价，缺少 provider/channel 成本价快照会导致毛利、成本审计和 fallback 成本分析不完整；进入成本报表或多 channel 商业化前；增加 provider/channel cost price 与请求级 cost snapshot。
```

风险：

```text
当前只记录客户售卖价。
无法计算 provider/channel 成本、毛利，也无法比较 fallback channel 成本。
```

计划完善时机：

```text
进入成本报表或多 channel 商业化前。
```

建议优先级：

```text
P1
```

---

### 11. 价格生效区间可能重叠

位置：

```text
migrations/000012_create_prices.up.sql:24
```

新增 TODO：

```sql
-- TODO(阶段7/production): [GAP-7-010] prices 允许同一 model/currency/pricing_unit 出现重叠 enabled 生效窗口，后台改价时可能导致结算价格不确定；开放价格后台管理前；用排他约束或事务化关停旧价格保证生效区间不重叠。
```

风险：

```text
同一模型可能出现多个 enabled 且生效时间重叠的价格。
当前 SQL 通过 ORDER BY 选最近一条，但商业账单上这不是强约束。
```

计划完善时机：

```text
开放价格后台管理前。
```

建议优先级：

```text
P1
```

---

### 12. Request / attempt 状态机缺少 SQL 守卫

位置：

```text
sql/queries/request_records.sql:54
```

新增 TODO：

```sql
-- TODO(阶段7/production): [GAP-7-003] request/attempt 状态更新目前没有状态机守卫，补偿任务或并发重试可能覆盖 succeeded/canceled 等终态；引入异步补偿或重复 settlement 前；为 SQL 增加合法前置状态条件并让终态更新具备幂等语义。
```

风险：

```text
当前 SQL 可以直接把 request 从任意状态更新为 running/failed/canceled/succeeded。
未来补偿任务或并发 settlement 可能覆盖终态。
```

计划完善时机：

```text
引入异步补偿或重复 settlement 前。
```

建议优先级：

```text
P1
```

---

### 13. 非流式请求调用上游前没有余额预检 / 预授权

位置：

```text
internal/service/gateway/chat_completion.go:65
```

新增 TODO：

```go
// TODO(阶段7/production): [GAP-7-001] 非流式请求调用上游前没有余额预检或预授权，余额不足用户可能先产生平台上游成本再在 settlement 阶段失败；公开计费 API 前；引入余额 preflight 或 pre-authorize，并在 settlement 成功后确认扣费。
```

风险：

```text
用户余额不足时，平台仍可能先请求上游并产生上游成本。
settlement 阶段虽然会防止余额扣成负数，但平台已经垫付成本。
```

计划完善时机：

```text
公开计费 API 前。
```

建议优先级：

```text
P0
```

---

### 14. 流式请求调用上游前没有预授权

位置：

```text
internal/service/gateway/chat_stream.go:105
```

新增 TODO：

```go
// TODO(阶段7/production): [GAP-7-002] 流式请求调用上游前没有预授权，长输出或恶意断开可能让平台先承担上游成本；公开 stream 计费 API 前；基于 max_tokens/模型价格冻结余额，拿到 final usage 后 settle，多余部分 refund。
```

风险：

```text
stream 输出长度不确定。
用户可能余额不足、长输出、主动断开，导致平台承担上游成本。
```

计划完善时机：

```text
公开 stream 计费 API 前。
```

建议优先级：

```text
P0
```

---

### 15. Request error message 可能泄漏内部细节

位置：

```text
internal/service/gateway/chat_request_record.go:94
```

新增 TODO：

```go
// TODO(阶段7/production): [GAP-7-005] request_records.error_message 当前保存原始内部错误，未来后台暴露请求日志时可能泄漏上游路径、配置细节或敏感上下文；开放请求日志查询前；区分 safe_user_message、internal_error_detail，并对后台展示做脱敏。
```

风险：

```text
当前 request_records.error_message 直接保存 err.Error()。
未来后台或日志查询暴露时，可能泄漏上游地址、配置细节或内部错误结构。
```

计划完善时机：

```text
开放请求日志查询前。
```

建议优先级：

```text
P2
```

---

### 16. Settlement recovery 与请求级幂等

位置：

```text
internal/service/gateway/chat_settlement.go
internal/service/gateway/chat_settlement_recovery.go
internal/app/workers/settlement_recovery_worker.go
```

状态：

```text
已完成，GAP-7-007 已关闭。
```

风险：

```text
上游成功且有可靠 usage 后，首次 settlement 失败不能 release 冻结余额。
```

已完成方案：

```text
settlement 已按 request_record_id 做成功重放一致性检查。
gateway 成功拿到可靠 usage 后先持久化 settlement_recovery_jobs。
首次 settlement 失败后由 worker claim job 并复用幂等 settlement 重试。
成功标记 succeeded，失败退避重试，耗尽后标记 dead 等人工处理。
```

建议优先级：

```text
已关闭
```

---

### 17. usage source 无法区分非流式与 stream

位置：

```text
internal/service/gateway/chat_settlement.go:106
```

新增 TODO：

```go
// TODO(阶段7/production): [GAP-7-008] usage_records.source 当前无法区分非流式 response 和 stream final usage，会降低账单审计与异常排查精度；收口 stream billing 报表前；在 ChatSettlementParams 中显式传入 usage source。
```

风险：

```text
usage_records 表支持 upstream_response / upstream_stream。
但 ChatSettlementParams 没有传 source，当前统一写 upstream_response。
```

计划完善时机：

```text
收口 stream billing 报表前。
```

建议优先级：

```text
P1
```

---

### 18. Ledger 缺少 pre-authorize / capture / refund 语义

位置：

```text
internal/core/ledger/service.go:33
```

新增 TODO：

```go
// TODO(阶段7/production): [GAP-7-011] ledger 当前只有 credit/debit，缺少 pre-authorize、capture、refund 的冻结/释放语义，stream 长输出和无 final usage 中断无法生产级控损；公开计费 API 前；引入余额预授权表或 reservation ledger，并实现 settle/refund 补偿流程。
```

风险：

```text
ledger 目前只有直接加款和直接扣款。
缺少请求开始前冻结余额、完成后确认扣费、失败后释放余额的语义。
```

计划完善时机：

```text
公开计费 API 前。
```

建议优先级：

```text
P0
```

---

### 19. 外部事务内 debit 幂等冲突处理不完整

位置：

```text
internal/core/ledger/service.go:184
```

新增 TODO：

```go
// TODO(阶段7/production): [GAP-7-012] 外部事务内并发使用同一 debit 幂等键时，CreateLedgerEntry 唯一冲突会使调用方事务失败且无法在当前事务内安全查询既有流水；引入并发 settlement/补偿任务前；使用请求级锁或 insert-first 幂等策略让外层事务可稳定重入。
```

风险：

```text
DebitWithQueries 复用外层事务。
如果并发 settlement 使用同一个幂等键，唯一约束冲突会让外层事务失败。
当前无法像独立 Debit 那样回滚后查询既有流水。
```

计划完善时机：

```text
引入并发 settlement / 补偿任务前。
```

建议优先级：

```text
P1
```

---

## 最重要的生产风险

### P0：公开计费 API 前必须处理

```text
1. 非流式请求前置余额检查 / 预授权。
2. 流式请求预授权、settle、refund。
3. ledger 增加 pre-authorize / capture / refund 语义。
4. settlement 请求级幂等完成检测。
```

原因：

```text
这些问题直接影响平台成本、重复扣费、用户余额和账单可信度。
```

---

### P1：进入后台管理 / 多项目 / 商业报表前必须处理

```text
1. API Key 创建授权与审计。
2. chat request 深度校验。
3. adapter contract 承载或拒绝 OpenAI-compatible 可选参数。
4. /v1/models 和 routing 接入 project policy。
5. provider/channel cost snapshot。
6. price 生效区间不重叠。
7. request/attempt 状态机守卫。
8. usage source 区分 stream / non-stream。
9. 外部事务内 debit 幂等冲突处理。
```

原因：

```text
这些问题会影响客户隔离、计费准确性、后台管理可靠性和审计可解释性。
```

---

### P2：生产部署质量项

```text
1. X-Request-ID 输入约束。
2. Redis 限流故障降级策略。
3. JSON body 严格校验。
4. OpenAI stream parser 替换 Scanner。
5. request error_message 脱敏分层。
```

原因：

```text
这些问题不一定马上造成账务错误，但会影响公网安全性、稳定性和可观测性。
```

---

## 建议下一步

建议不要直接进入第 8 阶段。

更合理的下一步是补一个第 7 阶段追加小节：

```text
7.17 Pre-authorization 设计与最小落地
```

目标不是一次做完复杂钱包系统，而是先把生产边界定对：

```text
1. 定义 pre-authorize / capture / refund 的领域模型。
2. 决定冻结余额是用独立 reservations 表，还是 ledger entry 扩展。
3. 非流式请求先做余额 preflight 或最小预授权。
4. stream 请求按 max_tokens / 模型价格冻结预算。
5. final usage 后 capture，少用则 refund。
6. 无 final usage 的取消/中断进入释放或风控策略。
```

建议优先实现顺序：

```text
1. 设计 user_balance_reservations 表或 ledger reservation 模型。
2. 增加 billing quote 能力：根据 max_tokens 和价格估算最大冻结金额。
3. gateway 在 adapter 调用前执行 pre-authorize。
4. settlement 成功后 capture。
5. adapter 调用失败 / 无 final usage 策略下 refund 或 release。
```

---

## AGENTS.md 修改建议

建议在 AGENTS.md 的阶段验收规则中新增：

```text
阶段验收前，必须输出生产欠账审计报告。

报告必须包含：

- 当前阶段已实现能力。
- 当前阶段仍保留的 production TODO。
- 每个 TODO 的风险、触发时机、未来替换方向。
- 哪些 TODO 阻塞进入下一阶段。
- 哪些 TODO 可以延后，但必须在公开生产前完成。
- 验证命令和结果。
```

建议在第 7 阶段计费原则中新增：

```text
第 7 阶段验收不能只检查 post-settle debit。
必须检查请求前余额保护或明确 pre-authorize 欠账。
如果尚未实现 pre-authorize，必须在代码中留下 production TODO，
并在交接文档中说明公开计费 API 前必须完成。
```

---

## 本次审计结论

第 7 阶段当前状态：

```text
功能闭环：基本完成
生产计费闭环：未完成
```

核心原因：

```text
已经具备 usage -> price snapshot -> billing -> ledger debit 的后置结算能力。
但还缺请求前余额保护、预授权、请求级 settlement 幂等、成本价快照等生产商业能力。
```

因此，当前项目不应以“第 7 阶段完全生产可用”收尾。

建议下一轮讨论重点：

```text
是否在进入第 8 阶段前，增加 7.17 pre-authorize 最小生产设计。
```
