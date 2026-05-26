package requestlog

import "errors"

// ErrInvalidStateTransition 表示 request 或 attempt 状态转移不符合状态机规则。
var ErrInvalidStateTransition = errors.New("requestlog invalid state transition")
