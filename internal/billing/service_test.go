package billing

import (
	"errors"
	"math/big"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

// numeric 创建测试用 pgtype.Numeric，value 表示 Int，exp 表示 10 的指数。
func numeric(value int64, exp int32) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(value), Exp: exp, Valid: true}
}

// nullNumeric 创建测试用 SQL NULL NUMERIC。
func nullNumeric() pgtype.Numeric {
	return pgtype.Numeric{Valid: false}
}

// defaultPriceSnapshot 返回一份支持 token_v1 的基础价格快照。
func defaultPriceSnapshot() PriceSnapshot {
	return PriceSnapshot{
		Currency:             "USD",
		PricingUnit:          PricingUnitPer1MTokens,
		InputPrice:           numeric(2_0000000000, -10),
		OutputPrice:          numeric(8_0000000000, -10),
		CachedInputPrice:     numeric(5000000000, -10),
		ReasoningOutputPrice: numeric(12_0000000000, -10),
		FormulaVersion:       FormulaVersionV1,
	}
}

// assertNumeric 校验 NUMERIC 的内部整数和小数指数，避免 float64 精度误差。
func assertNumeric(t *testing.T, got pgtype.Numeric, wantInt int64, wantExp int32) {
	t.Helper()

	if !got.Valid {
		t.Fatal("expected valid numeric")
	}
	if got.Exp != wantExp {
		t.Fatalf("expected numeric exponent %d, got %d", wantExp, got.Exp)
	}
	if got.Int == nil {
		t.Fatal("expected numeric int")
	}
	if got.Int.Cmp(big.NewInt(wantInt)) != 0 {
		t.Fatalf("expected numeric int %d, got %s", wantInt, got.Int.String())
	}
}

// TestCalculateChargesTokenClassesSeparately 验证普通、缓存和 reasoning token 会按各自价格计费。
func TestCalculateChargesTokenClassesSeparately(t *testing.T) {
	settlement, err := (Service{}).Calculate(Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
		CachedTokens:     200,
		ReasoningTokens:  100,
	}, defaultPriceSnapshot())
	if err != nil {
		t.Fatalf("calculate settlement: %v", err)
	}

	if settlement.Currency != "USD" {
		t.Fatalf("expected USD currency, got %q", settlement.Currency)
	}
	if settlement.FormulaVersion != FormulaVersionV1 {
		t.Fatalf("expected formula version %q, got %q", FormulaVersionV1, settlement.FormulaVersion)
	}

	// (800*2 + 200*0.5 + 400*8 + 100*12) / 1_000_000 = 0.0061000000。
	assertNumeric(t, settlement.Amount, 61_000000, -10)
}

// TestCalculateFallsBackToBasePricesForNullSpecialPrices 验证可空特殊价格未配置时回退到普通输入/输出价格。
func TestCalculateFallsBackToBasePricesForNullSpecialPrices(t *testing.T) {
	price := defaultPriceSnapshot()
	price.CachedInputPrice = nullNumeric()
	price.ReasoningOutputPrice = nullNumeric()

	settlement, err := (Service{}).Calculate(Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
		CachedTokens:     200,
		ReasoningTokens:  100,
	}, price)
	if err != nil {
		t.Fatalf("calculate settlement: %v", err)
	}

	// (800*2 + 200*2 + 400*8 + 100*8) / 1_000_000 = 0.0060000000。
	assertNumeric(t, settlement.Amount, 60_000000, -10)
}

// TestCalculateDefaultsEmptyFormulaVersion 验证旧数据未显式写公式版本时默认按 token_v1 计算。
func TestCalculateDefaultsEmptyFormulaVersion(t *testing.T) {
	price := defaultPriceSnapshot()
	price.FormulaVersion = ""

	settlement, err := (Service{}).Calculate(Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}, price)
	if err != nil {
		t.Fatalf("calculate settlement: %v", err)
	}

	if settlement.FormulaVersion != FormulaVersionV1 {
		t.Fatalf("expected default formula version %q, got %q", FormulaVersionV1, settlement.FormulaVersion)
	}
}

// TestCalculateRoundsToAmountScale 验证金额会四舍五入到 NUMERIC(20,10) 对应的小数位。
func TestCalculateRoundsToAmountScale(t *testing.T) {
	price := defaultPriceSnapshot()
	price.InputPrice = numeric(5, -5)
	price.OutputPrice = numeric(0, 0)
	price.CachedInputPrice = nullNumeric()
	price.ReasoningOutputPrice = nullNumeric()

	settlement, err := (Service{}).Calculate(Usage{
		PromptTokens: 1,
		TotalTokens:  1,
	}, price)
	if err != nil {
		t.Fatalf("calculate settlement: %v", err)
	}

	// 0.00005 / 1_000_000 = 0.00000000005，保留 10 位后四舍五入为 0.0000000001。
	assertNumeric(t, settlement.Amount, 1, -10)
}

// TestEstimateAuthorizationAmountUsesHigherCompletionPrice 验证冻结金额按 output/reasoning 中更贵的 completion 价格计算。
func TestEstimateAuthorizationAmountUsesHigherCompletionPrice(t *testing.T) {
	settlement, err := (Service{}).EstimateAuthorizationAmount(AuthorizationEstimate{
		PromptTokens:        1000,
		MaxCompletionTokens: 500,
	}, defaultPriceSnapshot())
	if err != nil {
		t.Fatalf("estimate reservation amount: %v", err)
	}

	if settlement.Currency != "USD" {
		t.Fatalf("expected USD currency, got %q", settlement.Currency)
	}
	if settlement.FormulaVersion != FormulaVersionV1 {
		t.Fatalf("expected formula version %q, got %q", FormulaVersionV1, settlement.FormulaVersion)
	}

	// (1000*2 + 500*12) / 1_000_000 = 0.0080000000。
	assertNumeric(t, settlement.Amount, 80_000000, -10)
}

// TestEstimateAuthorizationAmountFallsBackToOutputPrice 验证 reasoning 价格未配置时按普通 output 价格估算。
func TestEstimateAuthorizationAmountFallsBackToOutputPrice(t *testing.T) {
	price := defaultPriceSnapshot()
	price.ReasoningOutputPrice = nullNumeric()

	settlement, err := (Service{}).EstimateAuthorizationAmount(AuthorizationEstimate{
		PromptTokens:        1000,
		MaxCompletionTokens: 500,
	}, price)
	if err != nil {
		t.Fatalf("estimate reservation amount: %v", err)
	}

	// (1000*2 + 500*8) / 1_000_000 = 0.0060000000。
	assertNumeric(t, settlement.Amount, 60_000000, -10)
}

// TestEstimateAuthorizationAmountRejectsInvalidEstimate 验证冻结金额估算不接受负数 token。
func TestEstimateAuthorizationAmountRejectsInvalidEstimate(t *testing.T) {
	tests := []struct {
		name     string
		estimate AuthorizationEstimate
	}{
		{
			name:     "negative prompt tokens",
			estimate: AuthorizationEstimate{PromptTokens: -1},
		},
		{
			name:     "negative max completion tokens",
			estimate: AuthorizationEstimate{MaxCompletionTokens: -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (Service{}).EstimateAuthorizationAmount(tt.estimate, defaultPriceSnapshot())
			if !errors.Is(err, ErrInvalidUsage) {
				t.Fatalf("expected ErrInvalidUsage, got %v", err)
			}
		})
	}
}

// TestCalculateRejectsInvalidUsage 验证 usage token 约束和数据库 usage_records 约束保持一致。
func TestCalculateRejectsInvalidUsage(t *testing.T) {
	tests := []struct {
		name  string
		usage Usage
	}{
		{
			name:  "negative prompt tokens",
			usage: Usage{PromptTokens: -1},
		},
		{
			name:  "total token mismatch",
			usage: Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 16},
		},
		{
			name:  "cached tokens exceed prompt tokens",
			usage: Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15, CachedTokens: 11},
		},
		{
			name:  "reasoning tokens exceed completion tokens",
			usage: Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15, ReasoningTokens: 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (Service{}).Calculate(tt.usage, defaultPriceSnapshot())
			if !errors.Is(err, ErrInvalidUsage) {
				t.Fatalf("expected ErrInvalidUsage, got %v", err)
			}
		})
	}
}

// TestCalculateRejectsInvalidPrice 验证价格快照缺少必填价格或含负数时会失败。
func TestCalculateRejectsInvalidPrice(t *testing.T) {
	validUsage := Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}

	tests := []struct {
		name  string
		price PriceSnapshot
	}{
		{
			name: "empty currency",
			price: func() PriceSnapshot {
				price := defaultPriceSnapshot()
				price.Currency = ""
				return price
			}(),
		},
		{
			name: "missing input price",
			price: func() PriceSnapshot {
				price := defaultPriceSnapshot()
				price.InputPrice = nullNumeric()
				return price
			}(),
		},
		{
			name: "negative output price",
			price: func() PriceSnapshot {
				price := defaultPriceSnapshot()
				price.OutputPrice = numeric(-1, 0)
				return price
			}(),
		},
		{
			name: "negative cached input price",
			price: func() PriceSnapshot {
				price := defaultPriceSnapshot()
				price.CachedInputPrice = numeric(-1, 0)
				return price
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (Service{}).Calculate(validUsage, tt.price)
			if !errors.Is(err, ErrInvalidPrice) {
				t.Fatalf("expected ErrInvalidPrice, got %v", err)
			}
		})
	}
}

// TestCalculateRejectsUnsupportedPricingUnit 验证不支持的计价单位会返回专门错误。
func TestCalculateRejectsUnsupportedPricingUnit(t *testing.T) {
	price := defaultPriceSnapshot()
	price.PricingUnit = "per_token"

	_, err := (Service{}).Calculate(Usage{PromptTokens: 10, TotalTokens: 10}, price)
	if !errors.Is(err, ErrUnsupportedPricingUnit) {
		t.Fatalf("expected ErrUnsupportedPricingUnit, got %v", err)
	}
}

// TestCalculateRejectsUnsupportedFormula 验证不支持的公式版本不会被错误地按 token_v1 计算。
func TestCalculateRejectsUnsupportedFormula(t *testing.T) {
	price := defaultPriceSnapshot()
	price.FormulaVersion = "token_v2"

	_, err := (Service{}).Calculate(Usage{PromptTokens: 10, TotalTokens: 10}, price)
	if !errors.Is(err, ErrUnsupportedFormula) {
		t.Fatalf("expected ErrUnsupportedFormula, got %v", err)
	}
}
