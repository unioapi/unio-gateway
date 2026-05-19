# Phase 9 Plan - 后台管理

## 目标

提供后台管理能力，让 user、project、API key、provider、channel、model、price、request logs、billing logs 可以被安全管理。

阶段 9 前置必须完成 credential resolver 生产化。

## 任务

<a id="task-9-01-admin-auth"></a>
### TASK-9.01 Admin auth

状态：planned

范围：

1. 后台登录。
2. JWT。
3. 管理员权限模型。
4. 审计日志。

<a id="task-9-02-credential-management"></a>
### TASK-9.02 Credential 管理

状态：planned

范围：

1. 保存 credential_ref。
2. 明文凭据不长期落库。
3. KMS/master key 或 secret manager 解析。
4. 凭据轮换和审计。

关联 GAP：

```text
GAP-6-001
```

<a id="task-9-03-provider-channel-admin"></a>
### TASK-9.03 Provider/channel/model/price 管理

状态：planned

范围：

1. 管理 provider。
2. 管理 channel。
3. 管理 model/channel_model。
4. 管理 price 和生效窗口。
5. 变更影响 routing 和 `/v1/models`，不要求重启。

<a id="task-9-04-request-billing-dashboard"></a>
### TASK-9.04 Request 与 billing dashboard

状态：planned

范围：

1. 查看 request records。
2. 查看 attempts。
3. 查看 usage、price snapshot、ledger。
4. 错误详情脱敏展示。
5. 支持按 user/project/api_key/model/provider/channel 查询。

