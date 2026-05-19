# Phase 6 Plan - 模型与渠道

## 目标

把 provider、channel、model、channel_model 和基础 routing 从代码常量推进到数据库业务数据。

本阶段要明确 Unio API 的核心产品边界：

1. Provider 是业务服务商，例如 OpenAI、Anthropic、Gemini。
2. Channel 是某个 provider 下的具体上游渠道，包含 base URL、credential_ref、优先级、健康状态和模型映射。
3. Model 是对外暴露的模型 ID，是 opaque string。
4. Adapter 是代码能力，不等于 provider，也不等于 channel。
5. Routing 负责为某个 project/model 选择可用 channel。

## 涉及文件

| 文件 | 作用 |
| --- | --- |
| [migrations/000003_create_model_channel_tables.up.sql](../../../migrations/000003_create_model_channel_tables.up.sql) | provider/channel/model/channel_model schema。 |
| [sql/queries/model_catalog.sql](../../../sql/queries/model_catalog.sql) | `/v1/models` 查询。 |
| [sql/queries/routing.sql](../../../sql/queries/routing.sql) | routing 候选查询。 |
| [internal/modelcatalog/catalog.go](../../../internal/modelcatalog/catalog.go) | model catalog service。 |
| [internal/routing/router.go](../../../internal/routing/router.go) | routing service。 |
| [internal/credential/resolver.go](../../../internal/credential/resolver.go) | credential_ref 解析边界。 |
| [internal/channel/runtime.go](../../../internal/channel/runtime.go) | 传给 adapter 的运行时 channel 参数。 |

## 任务

<a id="task-6-01-model-channel-schema"></a>
### TASK-6.01 Provider/channel/model schema

状态：done

目标：

```text
让模型、渠道和 provider 成为数据库业务数据，而不是写死在代码或 env。
```

已完成：

1. providers。
2. channels。
3. models。
4. channel_models。
5. enabled、priority、base_url、credential_ref、upstream_model。
6. model catalog 基础查询。
7. routing 基础查询。

关键约束：

1. `models.model_id` 是 opaque string，不按 `/` 等格式解析。
2. `channel_models.upstream_model` 表达真实上游模型名。
3. `channels.credential_ref` 只保存凭据引用，不保存长期明文 key。

<a id="task-6-02-credential-resolver"></a>
### TASK-6.02 Credential resolver 生产化

状态：deferred

目标：

```text
让 routing 能根据 credential_ref 安全解析出上游调用需要的明文 credential。
```

当前状态：

```text
第 6 阶段允许开发期 StaticResolver 占位。
```

生产形态：

1. credential_ref 指向密文记录、secret manager path 或 KMS 保护的数据。
2. resolver 根据 credential_ref 解析出明文 credential。
3. 明文 credential 只进入 `channel.Runtime`。
4. adapter 不知道 credential 存储、解密和轮换方式。
5. credential 轮换和读取必须有审计。

计划处理时机：

```text
阶段 9 前置小节，进入后台 channel 管理和凭据管理前。
```

关联 GAP：

- [GAP-6-001](../../production/TODO_REGISTER.md#gap-6-001)


<a id="task-6-03-bootstrap-wiring"></a>
### TASK-6.03 启动装配治理

状态：todo

目标：

```text
避免 main 函数膨胀成所有依赖装配和业务规则的堆叠点。
```

当前欠账：

1. [cmd/server/main.go](../../../cmd/server/main.go) 直接装配 credential、routing、adapter registry、gateway。
2. provider.adapter 是否存在于 adapter registry 缺少启动前校验。

计划实现：

1. 抽出 bootstrap/app wiring 组件。
2. `main` 只负责 config、logger、lifecycle、signal。
3. app wiring 负责依赖构造顺序。
4. 启动时校验数据库中 enabled provider 的 adapter key 是否存在。
5. 后台写入 provider.adapter 时也要校验 registry。

关联 GAP：

- [GAP-6-002](../../production/TODO_REGISTER.md#gap-6-002)
- [GAP-6-003](../../production/TODO_REGISTER.md#gap-6-003)


<a id="task-6-04-routing-policy"></a>
### TASK-6.04 Routing project policy

状态：todo

目标：

```text
让不同 project 能拥有不同模型可见性、预算、禁用和专属 channel 策略。
```

当前欠账：

1. routing 只校验 `project_id > 0`。
2. `/v1/models` 只看全局 enabled 模型和 channel。
3. 还没有 project_model/channel policy。

计划实现：

1. 设计 project 级模型可见性表或策略查询。
2. routing 选择 channel 前检查 project 是否允许使用该 model。
3. `/v1/models` 使用相同策略。
4. 后续预算策略进入 billing/preauthorization。
5. 专属 channel 或禁用 channel 也应进入 routing policy。

完成标准：

1. 用户看得到的模型一定可路由。
2. 用户不可用的模型不会出现在 `/v1/models`。
3. 同一模型可以因 project 不同而可见性不同。

关联 GAP：

- [GAP-6-005](../../production/TODO_REGISTER.md#gap-6-005)
- [GAP-6-006](../../production/TODO_REGISTER.md#gap-6-006)


<a id="task-6-05-routing-error-semantics"></a>
### TASK-6.05 Routing 错误语义

状态：todo

目标：

```text
让 gateway 能区分模型不存在、模型不可见、模型存在但无可用 channel、channel 配置错误。
```

计划实现：

1. 增加 model exists 查询。
2. 增加 project-visible model 查询。
3. routing candidate 查询为空时区分原因。
4. 定义 `ErrModelNotFound`、`ErrModelNotAvailable`、`ErrNoAvailableChannel`。
5. HTTP 层映射成安全 OpenAI-compatible error。

关联 GAP：

- [GAP-6-007](../../production/TODO_REGISTER.md#gap-6-007)


