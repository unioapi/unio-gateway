package billing

import "errors"

var (
	// ErrInvalidUsage 表示 usage token 数量不满足计费约束。
	ErrInvalidUsage = errors.New("billing: invalid usage")
	// ErrInvalidPrice 表示 price snapshot 缺少必需价格或价格无效。
	ErrInvalidPrice = errors.New("billing: invalid price")
	// ErrUnsupportedPricingUnit 表示当前 billing service 不支持该计价单位。
	ErrUnsupportedPricingUnit = errors.New("billing: unsupported pricing unit")
	// ErrUnsupportedFormula 表示当前 billing service 不支持该价格计算公式。
	ErrUnsupportedFormula = errors.New("billing: unsupported formula")
)
