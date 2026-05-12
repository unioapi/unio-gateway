package channel

import "time"

// Runtime 表示一次 adapter 调用使用的运行时渠道参数。
// TODO(阶段6/production): 后续这些值会由 routing 从数据库 channel 业务数据中选择并解密得到，避免生产请求依赖硬编码 channel。
type Runtime struct {
	ID      int64
	BaseURL string
	APIKey  string
	Timeout time.Duration
}
