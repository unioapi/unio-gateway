package gateway

// NeverRetryClassifier 是保守的错误分类器，默认不重试任何错误。
type NeverRetryClassifier struct{}

// IsRetryable 始终返回 false，避免没有明确错误分类时误触发 fallback。
func (NeverRetryClassifier) IsRetryable(err error) bool {
	return false
}
