# Phase 6 Plan - 模型与渠道

## 目标

把 provider、channel、model、channel_model 和基础 routing 从代码常量推进到数据库业务数据。

本阶段仍不做完整后台管理，但 schema 和 service 边界必须服务于未来后台动态管理。

## 任务

<a id="task-6-01-model-channel-schema"></a>
### TASK-6.01 Provider/channel/model schema

状态：done

范围：

1. 建立 providers。
2. 建立 channels。
3. 建立 models。
4. 建立 channel_models。
5. 支持 enabled、priority、base_url、credential_ref、upstream_model。

<a id="task-6-02-credential-resolver"></a>
### TASK-6.02 Credential resolver 生产化

状态：deferred

范围：

1. 当前允许开发期 static resolver。
2. 后续使用 KMS/master key、secret manager 或密文表解析 `credential_ref`。
3. adapter 只接收明文 runtime credential，不知道存储方式。

关联 GAP：

```text
GAP-6-001
```

<a id="task-6-03-bootstrap-wiring"></a>
### TASK-6.03 启动装配治理

状态：todo

范围：

1. 从 `main` 中抽出 bootstrap/app wiring。
2. 启动时校验 provider.adapter 是否存在于 adapter registry。
3. 保持 `main` 只负责配置、生命周期和退出信号。

关联 GAP：

```text
GAP-6-002
GAP-6-003
```

<a id="task-6-04-routing-policy"></a>
### TASK-6.04 Routing project policy

状态：todo

范围：

1. routing 不只校验 `project_id > 0`。
2. 引入 project 级模型可见性、预算、禁用、专属 channel 策略。
3. `/v1/models` 和 routing 共用同一套可见性规则。

关联 GAP：

```text
GAP-6-005
GAP-6-006
```

<a id="task-6-05-routing-error-semantics"></a>
### TASK-6.05 Routing 错误语义

状态：todo

范围：

1. 区分模型不存在。
2. 区分模型存在但无可用 channel。
3. 区分 channel credential 或配置错误。
4. gateway 映射成安全用户错误。

关联 GAP：

```text
GAP-6-007
```
