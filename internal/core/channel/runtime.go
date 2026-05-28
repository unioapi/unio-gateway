package channel

import "time"

// Runtime 表示一次 adapter 调用使用的运行时渠道参数。
type Runtime struct {
	ID      int64
	BaseURL string
	APIKey  string
	Timeout time.Duration
}
