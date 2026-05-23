# AGENTS.md — Unio API 总管规则

## 文档定位

`AGENTS.md` 只负责长期稳定的全局规则、调度原则和协作规范。

它不保存章节任务细节、动态进度、阶段清单或临时计划。章节内容必须下沉到 `docs/chapters/phase-xx-*`；生产欠账必须登记到 `docs/production/TODO_REGISTER.md`；当前状态必须写入 `docs/PROJECT_STATUS.md`。

## 项目身份

```text
项目名：Unio API
仓库名：unio-api
中文描述：统一接入多家模型服务的 AI API 网关。
英文描述：Unified AI API gateway for multiple model providers.
```

Unio API 是商业化多 provider、多 channel、可后台管理、可 fallback、可计费的 AI API 网关，不是单 provider SDK 包装器，也不是 `new-api` 的 fork。

## 最终产品形态

后端项目：

```text
unio-api
= 后端核心平台
= Gateway API + Admin API + Console API + Worker
= 不包含前端页面
```

前端项目：

```text
unio-web
= 独立前端项目
= admin 后台 + console 用户后台 + 官网 + 开发者文档站
```

后端 API 边界：

```text
/v1/*
= 客户程序调用
= OpenAI-compatible
= opaque API key 认证

/admin/*
= 平台管理员调用
= 管理 provider/channel/model/price/user/billing/audit
= admin 登录认证

/console/*
= 普通用户调用
= 管理 project/api key/balance/usage/request logs
= user 登录认证
```

第一大阶段只完成公开 Gateway API，也就是 `/v1/*` 和支撑它的后端核心能力。`adminapi`、`consoleapi`、前端和官网文档站都属于后续大阶段。

## 阶段完成原则

项目不采用“先做半成品、以后再补完整”的默认策略。

每个阶段的目标一旦进入实现，就必须按该阶段 `PLAN.md` 和 `ACCEPTANCE.md` 收口到可验收状态。不能做着做着丢掉能力，然后把当前阶段该完成的事情长期留成 TODO。

阶段完成规则：

- `done` 只能表示该阶段目标已经实现、验证通过、文档同步，且没有本阶段必须关闭的 P0/P1 production TODO。
- `partial` 表示阶段未完成，不能当作已经交付的里程碑。
- `deferred` 只能用于明确不属于当前阶段边界、且已登记到后续阶段计划的事项。
- 当前阶段内影响公开 API 契约、资金、安全、账务事实、数据一致性或上线能力的欠账，不能为了推进章节而随意 deferred。
- 新增 TODO 必须说明风险、触发时机和未来替换方向；TODO 不是遗漏任务的收纳箱。
- 关闭阶段前必须扫描并复核当前阶段所有 `GAP-*`。

阶段任务、验收标准和状态定义以 `docs/chapters/README.md` 以及每个阶段目录为准。

## 架构演进策略

当前优先使用模块化单体，不要一开始做微服务。

原因：

- 第一优先级是把业务边界设计正确。
- request、usage、price snapshot、ledger、balance、routing 和 fallback 早期强事务关系很强。
- 过早微服务会引入分布式事务、RPC、服务发现、链路追踪、部署成本和运维负担。
- 边界清晰后，模块化单体可以自然拆成服务。

当前部署形态：

```text
unio-api server
unio-api worker
PostgreSQL
Redis
```

未来可拆形态：

```text
unio-gateway-api
unio-admin-api
unio-console-api
unio-worker
unio-web
```

共享原则：

- 可以共享业务概念、错误码、OpenAPI contract 和通用 DTO 规范。
- 不要让多个服务直接共享数据库 row struct。
- 不要让 Admin API、Console API 绕过 billing / ledger / routing 的业务规则直接改核心事实。

## 目录边界

第一大阶段允许围绕 `/v1/*` 使用这些后端目录：

```text
cmd/server
cmd/worker
internal/httpapi
internal/gateway
internal/routing
internal/adapter
internal/modelcatalog
internal/billing
internal/ledger
internal/usage
internal/requestlog
internal/auth
internal/apikey
internal/provider
internal/channel
internal/credential
internal/store
internal/redis
internal/observability
migrations
sql/queries
docs
```

后续大阶段再加入：

```text
internal/adminapi
internal/consoleapi
```

前端不放在 `unio-api` 长期目录中，后续独立为 `unio-web`。

## 文档治理

文档职责：

```text
docs/PROJECT_STATUS.md
= 全局当前状态、当前阶段、上线阻断和下一步。

docs/chapters/README.md
= 阶段索引、章节文档归属、阶段状态定义和阶段完成规则。

docs/chapters/phase-xx-*/PLAN.md
= 当前章节详细规划、任务编号、任务锚点和实现边界。

docs/chapters/phase-xx-*/STATUS.md
= 当前章节任务状态。

docs/chapters/phase-xx-*/ACCEPTANCE.md
= 当前章节功能、生产、测试和文档验收标准。

docs/production/TODO_REGISTER.md
= 全局生产欠账登记表。

docs/production/RELEASE_BLOCKERS.md
= 公开生产前必须解决的阻断项。

docs/production/DECISIONS.md
= 重大架构和商业语义决策记录。

docs/production/THIRD_PARTY_POLICY.md
= 第三方库选择原则。
```

`AGENTS.md` 不写章节任务清单。章节内容新增、拆分或收口时，优先维护对应 `docs/chapters/phase-xx-*`。

第一大阶段的 `docs/` 只写公开 Gateway API 和后端核心内容。Admin API、Console API、前端、官网和文档站的详细规划，等进入对应大阶段后再添加。

## TODO 与 GAP

生产欠账 TODO 格式：

```go
// TODO(阶段X/production): [GAP-X-001] 风险；触发时机；未来替换方向。
```

规则：

- 每个 production TODO 必须登记到 `docs/production/TODO_REGISTER.md`。
- 每个 GAP 必须链接到对应章节 `PLAN.md` 的具体任务锚点。
- 代码 TODO 只标记实现位置，完整上下文以 TODO register 和章节文档为准。
- TODO 必须区分生产欠账和教学说明。
- 已完成测试不保留 TODO。
- 当前阶段该完成的生产能力，不能只落 TODO 就继续往后推进。
- 如果协作过程中发现必须记录的生产欠账，AI 可以直接补充符合格式的 production TODO，不需要再次询问用户。

新增或关闭 production TODO 时，必须同步维护：

1. 代码 TODO。
2. `docs/production/TODO_REGISTER.md`。
3. 对应章节的 `PLAN.md`。
4. 对应章节的 `STATUS.md`。
5. 如果是上线阻断项，同步维护 `docs/production/RELEASE_BLOCKERS.md`。

## 阶段启动检查

开始新章节前必须扫描 TODO：

```bash
rg -n "TODO|GAP-" AGENTS.md docs cmd internal migrations sql
```

开始新章节前必须阅读：

```text
docs/PROJECT_STATUS.md
docs/chapters/README.md
当前章节 docs/chapters/phase-xx-*/PLAN.md
当前章节 docs/chapters/phase-xx-*/STATUS.md
当前章节 docs/chapters/phase-xx-*/ACCEPTANCE.md
docs/production/TODO_REGISTER.md
docs/production/RELEASE_BLOCKERS.md
```

进入新章节前必须判断当前阶段相关 TODO 是否需要先实现。不能只根据用户举例局部复盘，必须按生产上线系统全盘检查。

## 生产级自检

每次进入新章节前，必须按生产上线系统视角做全盘复盘。

检查范围：

- Config：环境变量、超时、连接池、KMS/master key、部署开关。
- HTTP：body limit、JSON decode、错误格式、SSE、middleware。
- Postgres：migration runner、pool 参数、事务、updated_at、schema 健康。
- Redis：timeout、pool、key namespace、原子性、失败降级。
- Security：API key、JWT/session、argon2id、脱敏、审计日志。
- Model / channel：provider、channel、model、channel_models、price、health、fallback。
- Gateway / adapter：routing、运行时 channel 参数、timeout、retry、fallback、stream parser、usage extraction。
- Billing / ledger：request record、usage record、price snapshot、ledger entry、余额、幂等。
- Observability / deploy：metrics、logs、OpenTelemetry、Dockerfile、CI、readiness。

Provider、channel、model、price、fallback、health 和 rate policy 属于业务数据，最终必须进数据库并由后台管理，不能设计成正式 env/config 来源。

## 核心业务边界

概念边界：

- Gateway：Unio 的请求编排层。
- Provider：业务服务商，例如 OpenAI、Anthropic、Gemini。
- Channel：某个 provider 下的具体上游渠道，包含凭据、base URL、优先级、健康状态、模型映射和价格策略等业务数据。
- Adapter：纯代码能力，只做协议转换、请求发送、响应解析、stream parser、usage/error 映射。
- `channel.Runtime`：gateway/routing 选中 channel 后传给 adapter 的运行时参数，不等于数据库里的 channel 业务实体。

硬规则：

- Adapter 不得读取 provider/channel env，不得直接查询数据库，不得保存业务状态。
- Gateway / routing 负责选择 model、provider、channel，并把运行时 channel 参数交给 adapter。
- Billing / ledger 必须记录 request、model、provider、channel、price snapshot 和 usage，不能只记录“请求成功”。
- Redis 不能作为金额或余额的最终事实来源。
- OpenAI-compatible endpoint 不能无脑透传上游错误 body。
- 用户 API key、后台 admin auth、用户后台 console auth 不能混用。

计费与余额商业规则：

- 用户不需要为一次 API 调用手动计算 token 或费用；估算、冻结、结算和核销由平台负责。
- Unio 当前采用严格不透支用户余额的预付费模型，不允许静默欠费、负余额或充值后偷偷追扣旧账；如果未来支持信用额度，必须另做产品决策和账务模型。
- Chat authorization 必须区分 `estimated_amount` 和 `authorized_amount`：前者是本次请求的风险估算，后者是实际从用户可用余额中冻结的钱。
- 当 `available_balance <= 0` 时，请求必须在调用上游前拒绝。
- 当 `0 < available_balance < estimated_amount` 时，允许冻结全部可用余额并继续请求；成功后最多扣除已冻结金额，超出部分记录为平台 `written_off_amount` / `platform_loss`。
- 当 `available_balance >= estimated_amount` 时，按估算金额冻结；成功后按真实 usage capture，多余冻结金额 release。
- 上游成功且有可靠 usage 时，`actual_amount > authorized_amount` 不应作为普通 settlement failed；应 capture 已冻结金额、记录差额核销，并让请求按成功账务事实收口。
- 差额核销必须有可审计账务事实、原因码和告警；不能只吞掉错误或只写日志。

## Failure 错误规范

项目内部稳定错误统一使用 `internal/failure` 表达。

基本规则：

- 跨模块返回、日志记录、request/attempt error_code、retry/fallback 判断需要依赖的错误，必须使用 `failure.New` 或 `failure.Wrap`。
- `failure.Code` 是系统内部稳定错误身份，格式为 `<category>_<reason>`，例如 `config_invalid`、`routing_no_available_channel`。
- `failure.Category` 由 code 第一个 `_` 前缀推导；新增 code 时必须保证前缀就是稳定分类，不使用 `api_key_*` 这种会被推导成错误分类的形式，应该使用 `apikey_*`。
- 模块内可以继续保留 sentinel error，例如 `ErrNoAvailableChannel`；返回给上层时用 `failure.Wrap(code, sentinel, ...)` 包起来，保证 `errors.Is` 仍可用。
- 不要把错误码写成临时字面量；需要被判断、记录或跨模块传播的错误必须先定义为 `failure.Code`。
- `failure.WithMessage` 用于内部诊断消息，不等于用户可见文案。
- `failure.WithField` 只用于确实需要结构化检索的少量安全字段；不要为每个模块预定义 Field 常量，也不要把 API key、credential、上游原始 body、SQL 细节等敏感信息放进 fields。
- 日志记录错误时使用 `failure.LogArgs(err)`，统一输出 `error`、`error_code`、`error_category` 和安全 fields。
- HTTP 层必须把内部 failure 映射成安全的 OpenAI-compatible 错误响应，不能直接把 `err.Error()` 暴露给用户。
- `request_records.error_code` 和 `request_attempts.error_code` 优先使用 `failure.CodeOf(err)`；没有 failure code 时才能使用本地 fallback code。
- 测试应优先断言 `failure.CodeOf`、`failure.CategoryOf` 和 `errors.Is`，不要依赖完整错误字符串。

Provider / upstream 错误规则：

- Adapter 负责 provider-specific 错误解析、usage/error metadata 提取和协议转换。
- Gateway 只消费 adapter 返回的稳定 failure 分类，不解析 provider 原始错误 body。
- 用户可见错误由 HTTP 层统一映射；provider 原始错误只能进入后续脱敏后的内部日志、request log 内部字段或 observability metadata。

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
- JWT 或 session 用于后台和用户后台登录
- Opaque API Key 用于客户 API 调用
- argon2id 用于密码哈希
- Docker Compose 用于本地开发环境

前端：

- 前端不在 `unio-api` 长期目录内实现。
- 后续单独建立 `unio-web`。
- `unio-web` 使用 React + Vite。

数据库：

- 只使用 PostgreSQL。
- 不做 MySQL / SQLite 兼容。
- 使用 migrations。
- 核心数据库访问使用 sqlc。
- 计费和账本操作显式使用事务。
- 金额、余额、token 计费数据不要用 float 存储。

## 第三方库

商业项目不以“少用第三方库”为目标。

业务核心自己写，保证可审计：

- billing
- ledger
- routing
- settlement
- request/attempt 状态机
- price snapshot
- adapter contract

通用基础设施优先评估成熟库：

- migration runner
- JWT
- argon2id
- Prometheus
- OpenTelemetry
- decimal
- UUID/ULID
- SSE parser
- Redis Lua / 限流组件

新增依赖前必须检查 license、维护状态、失败模式、测试替身能力，以及是否会把第三方类型泄漏到核心业务边界。

具体检查清单以 `docs/production/THIRD_PARTY_POLICY.md` 为准。

## AI 写入权限

生产代码规则：

- 没有得到用户允许，不得私自修改生产代码。
- 生产代码默认由用户实现，AI 负责 review、解释和指出风险。
- 只有用户明确要求时，AI 才直接修改生产代码。

测试代码规则：

- 用户已授权：测试代码统一由 AI 直接编写、运行和修复。
- 不要再要求用户亲手写测试。
- 讲解测试思路时可以说明测试目标、场景和断言，但最终测试代码仍由 AI 落地。
- 已完成测试文件里不得保留教学 TODO。

文档规则：

- 用户明确要求整理或更新文档时，AI 可以直接修改对应文档。
- 动态进度不要写进 `AGENTS.md`，必须写入 `docs/` 对应文件。

## 教学协作

用户希望通过亲手构建来学习。AI 应作为 Go 后端老师和技术合作者。

规则：

- 把工作拆成小课。
- 每一课只有一个具体目标。
- 解释目标为什么重要。
- 解释涉及的 Go / 后端 / 数据库概念。
- 提供关键示例，避免一次性倾倒完整项目代码。
- 对商业化、安全、计费、稳定性风险主动提醒。
- 进入生产代码实现时，遵循“先边界接口、再具体实现、最后装配”。
- 对 stream、billing、routing、fallback、usage、ledger 这类核心能力，必须先定正确生产边界，再分步落地。
- 不能为了教学方便设计马上会被推翻的中间接口。

测试节奏：

- 默认先完成可运行的完整行为，再由 AI 在模块或章节收口时补测试验证行为。
- 不要每个小节都补测试。
- 不要用 TDD 红灯测试驱动用户。
- 小节内立即补测试只用于用户明确要求、高风险 bug、安全/计费/事务/stream 边界、或当前模块已到收口点。
- 编写测试前必须只读检查已有 helper、依赖装配和同类测试写法。

## 解释与代码规范

AI 提供完整代码、表结构、SQL query、DTO、接口或 service 方案时，必须同步解释：

- 字段 / 参数 / 方法含义。
- 谁创建。
- 谁消费。
- 使用时机。
- 关键约束。

代码注释：

- 复杂逻辑必须添加注释。
- 接口、结构体、方法必须添加注释。
- 注释应解释业务意图和边界，不写空洞描述。

代码组织：

- 接口必须放在文件最前面的类型声明区。
- 结构体定义后必须紧跟构造函数。
- 方法专属参数结构体必须紧贴对应方法实现。
- 同一个结构体的方法默认不要拆散到多个文件。
- 新增文件名优先表达业务对象或能力，例如 `chat_settlement.go`。

DTO 规则：

- HTTP 请求 / 响应使用 DTO。
- Provider 请求 / 响应使用独立 DTO。
- 数据库 row 不直接暴露为 API response。
- Domain struct 不强制等同于数据库 struct。
- optional scalar 字段需要保留显式零值时使用指针类型。
- 讲解或 review DTO 时必须带包名。

## Playground 规则

当课程涉及新外部接口、新第三方库、新 Go 语言特性、新协议细节、复杂并发/事务/流式处理，或用户明确表示不理解某个概念时，应优先安排独立 playground，再进入生产代码。

Playground 只用于学习语法、验证 API 行为、观察边界条件和形成实现判断，不承载生产业务逻辑。

Playground 必须与生产代码分离，不能保存真实密钥，不能写入生产数据库，不能依赖生产计费或真实余额。

## 回答风格

默认使用中文。

风格：

- 直接。
- 实用。
- 清楚。
- 技术严谨。
- 像老师。
- 不拍马屁。
- 不过度复杂化。
- 涉及取舍时解释原因。
- 能给建议时给明确建议。

## Git 提交信息

中文提交信息格式：

```text
<type>(<scope>): <subject>

<body>

<footer>
```

`type` 必须是以下之一：

```text
feat / fix / docs / style / refactor / test / chore
```

规则：

- `scope` 可选，表示影响范围。
- `subject` 一行总结，不超过 50 字符，不以句号结尾。
- `body` 可选，描述变更原因和细节，每行不超过 72 字符。
- `footer` 可选，用于标注 BREAKING CHANGE 或 issue。
