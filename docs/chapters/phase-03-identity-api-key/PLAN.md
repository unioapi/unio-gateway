# Phase 3 Plan - 用户与 API Key

## 目标

建立 customer API 调用身份边界，包括 user、project、API key、认证 middleware 和基础限流。

本阶段不实现后台管理 UI，但必须保证未来暴露 key 管理接口时有清楚授权和审计边界。

## 任务

<a id="task-3-01-identity-schema"></a>
### TASK-3.01 User、project、API key schema

状态：done

范围：

1. 建立 users、projects、api_keys。
2. API key 只保存 hash 和 prefix。
3. API key 归属 project，project 归属 user。

<a id="task-3-02-api-key-auth"></a>
### TASK-3.02 API key 认证 middleware

状态：partial

范围：

1. 从请求读取 bearer key。
2. hash 后查询数据库。
3. 将 user_id、project_id、api_key_id 放入 request context。
4. 控制 `last_used_at` 更新频率。

关联 GAP：

```text
GAP-3-001
```

<a id="task-3-03-api-key-management"></a>
### TASK-3.03 API Key 管理、禁用与审计

状态：todo

范围：

1. 支持 revoke/disable/list。
2. 创建 key 时校验调用者是否拥有 project。
3. 后台操作写入审计日志。
4. 创建后不再展示完整明文 key。

关联 GAP：

```text
GAP-3-002
GAP-3-007
```

<a id="task-3-04-rate-limit-production"></a>
### TASK-3.04 限流生产化

状态：todo

范围：

1. 默认限流阈值和窗口进入 config。
2. 后续支持 project/model/channel 级策略。
3. Redis INCR + EXPIRE 具备原子性。
4. Redis 故障时 fail-open/fail-closed 策略可配置。

关联 GAP：

```text
GAP-3-003
GAP-3-004
GAP-3-005
GAP-3-006
```

