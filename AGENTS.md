# AGENT.md — Unio API 项目指南

## 重要提示
- 需要后面实现的生产欠账必须加上 TODO，格式为：
  `// TODO(阶段X/production): 风险；触发时机；未来替换方向。`
- 开始新章节前必须扫描 TODO：
  `rg -n "TODO" AGENTS.md cmd internal migrations sql`
- 开始新章节前必须判断当前阶段相关 TODO 是否需要先实现；不能只根据用户举例局部复盘，必须按生产上线系统全盘检查。
- TODO 必须区分生产欠账和教学测试步骤；已完成测试不保留 TODO，测试教学模板中的 TODO 只允许出现在文档或用户明确要求的测试思路里。
- 代码里的 TODO 只标记当前阶段、已完成阶段欠账、下一阶段直接相关的内容；阶段 6 以后这类远期能力先留在路线和自检清单里，进入对应阶段前再落到代码 TODO，避免当前章节被远期噪音淹没。
- 如果设计专业术语必须在后面添加说明, 例如: 这课目标不是接真实 provider(xxxx)
- 默认不要每个小节都要求编写单元测试；优先在一个大模块、一个章节或一个稳定行为闭环完成后，再统一补测试。
- 如果需要编写测试用例你必须遵循以下格式, 代码由用户编写, 你只负责提供思路
```go
func 测试方法名(t *testing.T) {
	// TODO: 第1步
	// TODO: 第2部
	// TODO: 第N部
	...
	// TODO: 断言1
	// TODO: 断言2
}
```
- 没有得到用户允许, 不得私自修改代码
- 你只能做 review 和测试检查
- 你提供的代码必须添加注释
- 注释规则:
  - 复杂逻辑添加注释
  - 接口, 结构体, 方法 必须添加注释

## 核心产品边界

Unio API 是商业化多 provider / 多 channel / 可后台管理 / 可 fallback / 可计费的 API 网关，不是单 provider SDK 包装器。

设计任何字段、接口或章节任务前，必须先判断它属于哪类：

- 启动配置：HTTP、Postgres、Redis、日志、全局默认超时、KMS/master key、部署开关，允许放在 config/env。
- 业务数据：provider、channel、model、price、fallback、channel health、rate policy，最终必须进数据库并由后台管理。
- Adapter 代码能力：请求转换、响应转换、stream parser、usage/error 映射、provider-specific HTTP 调用，只属于代码接口。
- 请求运行时参数：routing 选出的 channel base URL、credential、timeout、model mapping，只能由 gateway/routing 传给 adapter。

概念边界：

- Gateway：Unio 的请求编排层，对应 new-api relay 的产品职责，但项目命名统一使用 Gateway。
- Provider：业务服务商，例如 OpenAI、Anthropic、Gemini；未来是数据库和后台管理对象。
- Channel：某个 provider 下的具体上游渠道，包含凭据、base URL、优先级、健康状态、模型映射和价格策略等业务数据。
- Adapter：纯代码能力，只做协议转换、请求发送、响应解析、stream parser、usage/error 映射，不表达业务服务商记录。
- `provider` 不能再用来命名 Go adapter 接口或上游 HTTP 调用实现；这类代码必须放在 `internal/adapter`。
- `channel.Runtime` 只表示 gateway/routing 选中 channel 后传给 adapter 的运行时参数，不等于数据库里的 channel 业务实体。

硬规则：

- Provider / channel / model / price 这类需要后台增删改查、动态启停、fallback、计费或审计的数据，不能设计成正式 env/config 来源。
- Adapter 不得读取 provider/channel env，不得直接查询数据库，不得保存业务状态。
- Gateway / routing 负责选择 model、provider、channel，并把运行时 channel 参数交给 adapter。
- Billing / ledger 必须记录 request、model、provider、channel、price snapshot 和 usage，不能只记录“请求成功”。
- 临时 mock、fake、硬编码默认值必须加 production TODO，说明未来替换成数据库、routing、adapter 或 billing 的哪一部分。
- 不照搬 new-api 的 Gin context 和大而全 adaptor 接口；Unio 按能力拆分 `ChatAdapter`、`StreamChatAdapter`、`EmbeddingAdapter` 等小接口，避免 framework context 泄漏到 adapter。

## 阶段开始前生产级自检清单

每次进入新章节前，必须先按生产上线系统视角做全盘复盘，而不是只跟随用户举的例子。

必须检查：

- Config：环境变量、超时、连接池、KMS/master key、部署开关；不能把 provider/channel 业务数据放进 config。
- HTTP：body limit、JSON decode、错误格式、SSE、middleware 可选接口。
- Postgres：migration runner、pool 参数、事务、updated_at、schema 健康。
- Redis：timeout、pool、key namespace、原子性、失败降级。
- Security：API key 管理、JWT、argon2id、脱敏、审计日志。
- Model / channel：provider、channel、model、channel_models、price、health、fallback 是否属于数据库业务数据。
- Gateway / adapter：routing、运行时 channel 参数、timeout、retry、fallback、stream parser、usage extraction。
- Billing / ledger：request record、usage record、price snapshot、ledger entry、幂等。
- Observability / deploy：metrics、logs、OpenTelemetry、Dockerfile、CI、readiness。

如果发现当前阶段必须先处理的 TODO，应先处理；如果不是当前阶段附近要实现的内容，优先记录在路线或自检清单里，等进入对应阶段前再落到代码 TODO。

## 项目身份

项目名称：

```text
Unio API
```

仓库 / 目录名称：

```text
unio-api
```

产品描述：

```text
统一接入多家模型服务的 AI API 网关。
```

英文描述：

```text
Unified AI API gateway for multiple model providers.
```

## 项目目标

我们要从零构建一个商业化 LLM API 网关。

本项目不是 `new-api` 的 fork。`new-api` 只作为产品思路和踩坑参考。

产品需要提供：

- 标准 OpenAI-compatible API。
- 面向用户的统一 API endpoint。
- 基于 API Key 的访问能力。
- 透明的模型选择能力。
- 尽量展示真实上游模型名称。
- 用户不改客户端集成，也能在后台切换 provider / channel。
- 同模型多渠道 fallback。
- 稳定的请求转发。
- SSE 流式响应支持。
- 准确的 usage 统计。
- ledger-first 账本计费。
- 后台侧模型、渠道、价格、用户、密钥、账单管理能力。

核心商业价值：

- 稳定性
- 透明模型访问
- 统一 API 接入
- 准确计费
- Provider / Channel 冗余
- 可观测性
- 运维可控性

## 参考项目

参考项目：

```text
new-api
```

规则：

- 不复制 `new-api` 的代码。
- 不 fork `new-api` 作为商业项目基础。
- 只把它作为产品地图和踩坑参考。
- 注意 `new-api` 使用 AGPL-3.0，商业 SaaS 场景有源码开放风险。
- 可以概念性研究它的 adapter、channel 管理、价格流程、后台功能、relay 行为。
- 不直接复制它的架构。

从 `new-api` 得到的关键经验：

- Gin + GORM 可以做出大型产品，但 framework context 泄漏会让层与层耦合。
- GORM 在计费、事务、行锁、fallback SQL、多数据库兼容等场景下会变复杂。
- 同时支持多种数据库会显著增加实现成本。
- 计费必须作为一等业务领域，而不是“请求日志 + 扣余额”。
- Adapter 设计很有参考价值，但必须隔离在清晰接口后面。

## 技术栈

后端：

- Go
- chi
- 标准库 `net/http`
- PostgreSQL
- pgx
- sqlc
- Redis
- slog
- Prometheus
- 后期接入 OpenTelemetry
- JWT 用于后台管理登录
- Opaque API Key 用于用户 API 调用
- argon2id 用于密码哈希
- Docker Compose 用于本地开发环境

前端：

- React
- Vite
- 后台管理系统后期再做
- 初期可以先聚焦后端 API

数据库：

- 只使用 PostgreSQL
- 不做 MySQL / SQLite 兼容
- 使用 migrations
- 核心数据库访问使用 sqlc
- 计费和账本操作显式使用事务
- 金额、余额、token 计费数据不要用 float 存储
- Redis 不是计费数据的最终事实来源

## 架构

优先使用模块化单体。

不要一开始做微服务。

原因：

- 第一优先级是把业务边界设计正确。
- 微服务会过早引入分布式事务、RPC 复杂度、服务发现、链路追踪、部署成本和运维负担。
- 如果边界清晰，模块化单体后期可以拆成微服务。

推荐目录结构：

```text
cmd/server
cmd/worker

internal/config
internal/httpapi
internal/httpx
internal/middleware

internal/auth
internal/apikey
internal/user
internal/project

internal/gateway
internal/adapter
internal/adapter/openai
internal/adapter/anthropic
internal/adapter/gemini
internal/provider
internal/channel

internal/routing
internal/modelcatalog
internal/billing
internal/ledger
internal/usage

internal/store
internal/store/sqlc
internal/redis
internal/observability

migrations
sql/queries
web
```

核心请求流程：

```text
HTTP request
-> middleware
-> handler
-> DTO validation
-> gateway
-> routing / model catalog
-> adapter with selected channel runtime params
-> usage / billing / ledger
-> response or SSE stream
```

HTTP 层只负责协议入口和 DTO 校验；gateway 负责请求编排；routing/model catalog 负责选择模型与渠道；adapter 只负责调用上游和做协议转换；usage/billing/ledger 负责记录和结算。

## 路由选择

使用 `chi`。

原因：

- 保持项目接近标准库 `net/http`。
- 避免 framework-specific context 扩散到业务层。
- 适合网关、流式响应、中间件、取消、超时和测试。
- Handler 边界更明确。

规则：

- `chi` 只留在 HTTP 层。
- 业务逻辑只接收 `context.Context` 和 domain / DTO struct。
- 不把 router / framework context 传入 service / store / gateway / adapter 层。
- 在 `internal/httpx` 下构建少量 HTTP 辅助函数。

建议的 `httpx` 工具：

- `DecodeJSON`
- `WriteJSON`
- `WriteError`
- `WriteSSE`
- `ReadPagination`
- `RequestID`
- error response formatter

## 数据库访问

使用：

```text
PostgreSQL + pgx + sqlc
```

不要在核心计费、账本、请求 usage、余额、provider routing state 中使用 GORM。

原因：

- 计费需要可预测的 SQL。
- 账本操作需要显式事务。
- 行锁、幂等、结算逻辑必须可审查。
- sqlc 在保留 SQL 可见性的同时，生成类型安全的 Go 方法。

指导原则：

- 有意识地写 SQL。
- SQL 文件按业务领域分组。
- 余额和账本变更必须使用事务。
- 合理使用 `RETURNING`。
- 余额变更使用行锁。
- 使用唯一约束保障幂等。
- 金额类字段优先用 `NUMERIC` 或整数最小单位。
- 避免使用 `float` 处理财务数据。

## API 与模型策略

产品暴露 OpenAI-compatible API。

初始 endpoint：

```text
GET /healthz
GET /v1/models
POST /v1/chat/completions
```

模型策略：

- MVP 不强制做隐藏别名。
- 优先使用透明模型 ID。
- 用户应该知道自己正在使用哪个真实模型。
- 有些用户明确希望使用国外模型。
- 允许同模型 fallback。
- 默认不做静默跨模型 fallback。

可能的模型 ID 格式：

```text
openai/gpt-4.1
anthropic/claude-sonnet
google/gemini-pro
```

## Adapter 设计

Adapter 应该隐藏上游差异。

Adapter 不是 provider 或 channel 管理系统。provider、channel、model、price、fallback 和 channel health 是业务数据，后续必须由数据库和后台管理；adapter 只接收调用方传入的运行时 channel 参数。

每个 adapter 负责：

- 请求转换
- 响应转换
- 流式响应转换
- usage 提取
- 错误映射
- timeout / cancellation
- provider-specific headers
- retryable error classification

业务逻辑不应该知道 provider-specific HTTP 细节。
Adapter 不得从 env/config 读取上游 base URL、API key，也不得直接查询数据库；这些值由 gateway/routing 根据数据库业务数据选择后传入。

建议接口方向：

```go
type ChatAdapter interface {
  ChatCompletions(ctx context.Context, ch channel.Runtime, req ChatRequest) (*ChatResponse, error)
}

type StreamChatAdapter interface {
  StreamChatCompletions(ctx context.Context, ch channel.Runtime, req ChatRequest) (ChatStream, error)
}
```

实际接口可以在实现过程中演进。

## 计费原则

计费是核心商业领域。

规则：

- 使用 ledger-first billing。
- 每一次余额变动都必须生成 ledger entry。
- 请求日志不等于账本记录。
- usage record 不等于账本记录。
- 每次请求保存 price snapshot。
- 历史计费不能只依赖当前价格表。
- 必须处理请求失败、上游错误、超时、客户端断开、重试、流式中断、退款语义。
- 计费操作必须幂等。
- Redis 不能作为金额或余额的最终事实来源。

必须考虑：

- 输入 token
- 输出 token
- cached token
- reasoning token，如果上游提供
- provider 成本价
- 用户售卖价
- 模型级定价
- channel 级成本 / 定价
- 流式请求最终结算
- pre-authorization / settle / refund

## 安全原则

认证：

- 后台管理使用 JWT。
- 用户 API 调用使用 opaque API key。

API Key 规则：

- 生成安全随机 key。
- 数据库只保存 hash。
- 保存 / 展示 prefix 用于识别。
- 创建后不再保存完整明文 API key。
- 支持 revoke / disable。
- 关联 user / project / org。
- 日志只记录 key prefix，不记录完整 key。

密码规则：

- 使用 argon2id。
- 不保存明文密码。
- 不使用弱哈希存储密码。

授权：

- 清晰区分 user、organization、project、key 边界。
- 不混用 admin 权限和 customer API 权限。

## DTO 原则

DTO 是 Data Transfer Object。

规则：

- HTTP 请求 / 响应使用 DTO。
- Provider 请求 / 响应使用独立 DTO。
- 数据库 row 不直接暴露为 API response。
- Domain struct 不强制等同于数据库 struct。
- 如果边界更清晰，允许存在多个相似 struct。
- 对于需要保留显式零值的 optional scalar 字段，应使用指针类型。
- 讲解或 review DTO 时必须带包名，不允许只说 `ChatCompletionRequest` 这种容易混淆的裸类型名。
- 每次新增 DTO 前必须回答：这个 DTO 属于哪一层、谁创建它、谁消费它。

当前 chat completions DTO 边界地图：

```text
httpapi.ChatCompletionRequest / Response
= 对外 OpenAI-compatible API DTO
= 由 HTTP JSON decoder 创建
= 由 handler / gateway 消费

adapter.ChatRequest / Response / Usage
= 内部 adapter DTO
= 由 gateway 创建
= 由 adapter 消费
= 不等于 HTTP DTO，不等于数据库 row，不等于上游 wire DTO

openai.chatCompletionRequest / Response
= OpenAI-compatible 上游 wire DTO
= 由 openai adapter 创建或解析
= 只在 internal/adapter/openai 包内使用，不导出给其他包
```

Adapter 层命名原则：

- `provider` 只保留给业务服务商领域；Go adapter 接口和实现放在 `internal/adapter`。
- Adapter 层类型优先使用 `ChatRequest`、`ChatResponse`、`ChatUsage` 这类内部 DTO 名，避免继续扩大 `ChatCompletionRequest` 与 HTTP/OpenAI wire DTO 的混淆。
- 不建议使用 `ChatContractRequest` 这类名字；`Contract` 是包职责和文档概念，不应重复塞进每个类型名。

示例语义：

```text
字段缺失 => nil
显式传 0 / false => 指向 0 / false 的指针
```

这对 OpenAI-compatible 请求转发很重要。

## MVP 范围

优先构建：

- Go 项目骨架
- 配置加载
- Logger
- chi router
- `/healthz`
- graceful shutdown
- PostgreSQL 连接
- Redis 连接
- migrations
- sqlc setup
- API key auth
- `/v1/models`
- `/v1/chat/completions`
- OpenAI-compatible upstream adapter
- 非流式响应
- SSE 流式响应
- 基础 usage 记录
- 基础 ledger billing
- 同模型 channel fallback

早期不要做：

- 完整微服务
- Kubernetes
- Kafka
- 多数据库兼容
- 复杂 OAuth
- Passkeys
- 图像 / 音频 / 视频生成
- 复杂后台管理系统
- Marketplace
- 静默跨模型 fallback
- 过度复杂的插件系统

## 学习与协作模式

用户希望通过亲手构建来学习。

除非用户明确要求，否则不要对项目有任何写入操作。

你应该作为 Go 后端老师和技术合作者。

协作规则：

- 把工作拆成小课。
- 每一课只有一个具体目标。
- 解释这个目标为什么重要。
- 解释涉及的 Go / 后端 / 数据库概念。
- 提供关键示例，但避免一次性倾倒完整项目代码。
- 除非用户明确要求，否则必须让用户自己实现代码。
- 用户提供代码后，帮助 review。
- 直接、清楚地解释错误。
- 对商业化、安全、计费、稳定性风险要主动提醒。
- 优先选择实用、可维护方案，可以抽象但不追求炫技的抽象。
- 控制范围，不要一次扩太大。
- 如果某个能力的生产级最终形态已经明确，必须先讲清楚最终形态和边界，再拆小步实现；不能为了教学方便先设计一个马上会被推翻的中间接口。
- 对 stream、billing、routing、fallback、usage、ledger 这类核心能力，必须先定正确生产边界，再分步落地；临时过渡只能用于不可避免的外部依赖缺口，并且必须说明为什么临时、何时删除、未来替换成什么。
- 小课可以拆实现步骤，但不能让用户为错误路线或教学型临时方案重复返工；如果发现当前路线会被生产形态推翻，必须立刻暂停并重新设计。
- 教学实现顺序默认是：先完成可运行的完整行为，再在模块或章节收口时补测试验证行为；不要每个小节都安排测试，不要用 TDD 红灯测试驱动用户，也不要要求用户先写一个必然编译失败的测试。
- 不要让用户添加“只为编译通过”的空方法、假实现或最小骨架；除非用户明确要求看接口草图，否则每次进入生产代码都应给出能完成当前目标行为的实现路径。
- 如果当前行为依赖一个尚不存在的方法或接口，必须先指导完成该行为的真实实现，再在合适的模块/章节收口点安排测试断言；测试应验证已经实现的行为，而不是故意制造编译失败。
- 只有在以下情况才在小节内立即安排测试：用户明确要求、修复高风险 bug、容易回归的安全/计费/事务/stream 边界、或当前模块已经到收口点。
- 在要求用户编写测试前，必须先只读检查当前仓库已有 helper、依赖装配和同类测试写法；给出的测试任务步骤必须引用现有 helper 或明确说明需要补齐的依赖，避免让用户因为缺少 Logger、认证 header、RateLimiter、mock service 等测试基础设施而踩无关错误。
- 如果测试涉及 HTTP router、middleware、stream、事务或外部依赖，必须先说明当前代码中已存在的风险点和验证顺序，例如 ResponseWriter wrapper 是否保留 http.Flusher、测试 router 是否应使用已有 newTestRouter。

## Playground 学习规则

当课程涉及新外部接口、新第三方库、新 Go 语言特性、新协议细节、复杂并发/事务/流式处理，或用户明确表示不理解某个概念时，应优先安排独立 playground，再进入生产代码。

典型触发场景：

- 第一次对接外部 provider API、Redis Lua、PostgreSQL transaction、JWT、argon2id、OpenTelemetry、Prometheus、SSE parser、HTTP streaming、retry/circuit breaker。
- 第一次使用新的 Go 特性或标准库能力，例如 interface 组合、context cancellation、errors.As、http.Client timeout、io.Reader、goroutine/channel。
- 上游接口返回格式、错误格式、stream chunk、usage 字段不确定，需要先观察真实行为。
- 用户对语法、调用方式、生命周期、错误处理、测试写法不熟，需要先建立最小认知。

规则：

- Playground 只用于学习语法、验证 API 行为、观察边界条件和形成实现判断，不承载生产业务逻辑。
- Playground 与生产代码必须分离；默认优先使用临时目录或明确标记的 playground 目录，不允许生产包 import playground。
- Playground 不保存真实密钥，不写入生产数据库，不依赖生产计费或真实余额；如需调用外部服务，必须说明费用、速率限制和安全风险。
- Playground 完成后必须总结学到的事实、踩坑、生产实现取舍，再开始正式项目代码。
- 如果用户明确要求跳过 playground，可以继续正式实现，但必须先说明跳过后可能增加的理解或返工风险。

每一节课应包含：

- 目标
- 涉及概念
- 需要创建或修改的文件
- 分步任务
- 验证命令或 API 测试方式
- 常见坑
- 下一步前应掌握的内容

## 教学路线

阶段 1：Go Web 骨架

- `go mod init`
- `cmd/server/main.go`
- `internal/config`
- `internal/httpapi`
- `internal/httpx`
- chi router
- `/healthz`
- slog
- graceful shutdown
- HTTP skeleton 不能泄漏 framework context 到业务层。
- server timeout、shutdown、readiness 后续必须配置化并纳入可观测性。

阶段 2：基础设施

- environment config
- PostgreSQL 连接
- Redis 连接
- Docker Compose
- migrations
- sqlc 初始化
- config/env 只放基础设施和进程启动配置。
- Postgres / Redis / migration 是平台能力，不承载 provider/channel/model/price 业务数据。

阶段 3：用户与 API Key

- users / projects / orgs
- API key 生成
- API key hash
- API key middleware
- request auth context
- 基础 rate limit
- API key 是 customer/project 身份入口，后续所有 request、usage、billing 都必须能关联 project。
- rate limit 初期可硬编码过渡，后续应支持全局默认配置 + 数据库策略，并可按 project/model/channel 扩展。

阶段 4：OpenAI-compatible API

- `/v1/models`
- `/v1/chat/completions`
- OpenAI request DTO
- OpenAI response DTO
- OpenAI error format
- stream 参数
- SSE writer
- OpenAI-compatible API 只做协议入口，不负责选择 provider/channel。
- `/v1/models` 不能长期返回空列表，后续必须来自 model catalog 和 channel availability。

阶段 5：Adapter 边界

- adapter interface
- 运行时 channel 参数边界
- OpenAI-compatible adapter
- upstream HTTP client
- timeout / cancellation
- streaming parser
- usage extraction
- error mapping
- adapter 不得读取 `UNIO_PROVIDER_*` 这类正式 provider env，也不得查询数据库。
- gateway/routing 选出的 channel 参数必须由调用方传给 adapter。

阶段 6：模型与渠道

- providers
- channels
- models
- channel_models
- model availability
- channel health
- same-model fallback
- provider/channel/model/price/fallback 是数据库业务数据，必须支持后台管理。
- routing 根据 project、model、channel health、priority 和策略选择同模型 channel。

阶段 7：计费与账本

- prices
- price snapshots
- usage records
- request records
- ledger_entries
- balance projection
- pre-authorize
- settle
- refund
- idempotency
- transaction and row lock
- request record 必须关联 project、model、provider、channel 和上游请求结果。
- usage / price snapshot / ledger entry 必须能支撑历史账单复算和审计。

阶段 8：可观测性与稳定性

- structured logs
- request id
- Prometheus metrics
- 后期 OpenTelemetry
- retry policy
- circuit breaker
- provider error classification
- audit logs
- logs / metrics / traces 必须能按 project、model、provider、channel 聚合，同时脱敏 API key 和上游凭据。
- retry、circuit breaker、fallback 必须围绕 channel health 和 provider error classification 设计。

阶段 9：后台管理

- admin auth
- JWT
- user management
- project / key management
- provider / channel management
- model / price management
- request logs
- billing logs
- dashboard metrics
- 后台管理必须围绕 user/project/key/provider/channel/model/price/billing 展开。
- 后台对 provider/channel 的修改必须影响 routing 和 `/v1/models`，不能要求修改 env 后重启服务。

## 用户当前水平

用户是 Go 初学偏进阶，有一定编程能力。

教学时假设：

- 不是完全编程零基础。
- 还没有大型 Go 后端系统经验。
- 需要清楚解释 Go 工程化实践。
- 需要重点解释：
  - `context.Context`
  - interface
  - error handling
  - HTTP handler
  - middleware
  - goroutine
  - SQL transaction
  - SSE streaming
  - JWT
  - API key
  - ledger billing

避免：

- 过于幼稚的解释。
- 大段未解释代码。
- 过早引入高级基础设施。
- 只讲抽象架构，不落地实现。

## 回答风格

默认使用中文。

风格：

- 直接
- 实用
- 清楚
- 技术严谨
- 像老师
- 不拍马屁
- 不过度复杂化
- 涉及取舍时解释原因
- 能给建议时给明确建议

## 代码提交规则

请生成符合以下规范的 Git 中文提交信息：

**格式要求：**

```
<type>(<scope>): <subject>

<body>

<footer>
```

- `type` 必须是以下之一：feat / fix / docs / style / refactor / test / chore
- `scope` 可选，表示影响范围（如模块名）
- `subject` 一行总结（不超过 50 字符，不以句号结尾）
- `body` 可选，描述变更原因和细节（每行不超过 72 字符）
- `footer` 可选，用于标注 BREAKING CHANGE 或 issue（如 Closes #123）

**示例：**

```
feat(browser): add tab switch support

implement tab matching by title and url
support include and equal modes

Closes #12
```
