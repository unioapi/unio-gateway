package ledger

import (
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
)

// sameNumeric 比较两个 pgtype.Numeric 是否表示同一个金额值。
func sameNumeric(left pgtype.Numeric, right pgtype.Numeric) bool {
	leftRat, leftOK := numericRat(left)
	rightRat, rightOK := numericRat(right)
	if !leftOK || !rightOK {
		return leftOK == rightOK
	}

	return leftRat.Cmp(rightRat) == 0
}

// numericRat 将 pgtype.Numeric 转成有理数，用于金额等值比较。
func numericRat(value pgtype.Numeric) (*big.Rat, bool) {
	if !value.Valid || value.NaN || value.InfinityModifier != pgtype.Finite || value.Int == nil {
		return nil, false
	}

	rat := new(big.Rat).SetInt(new(big.Int).Set(value.Int))
	if value.Exp > 0 {
		rat.Mul(rat, new(big.Rat).SetInt(pow10(value.Exp)))
	}
	if value.Exp < 0 {
		rat.Quo(rat, new(big.Rat).SetInt(pow10(-value.Exp)))
	}

	return rat, true
}

// pow10 返回 10 的 exp 次方。
func pow10(exp int32) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil)
}

// isPositiveNumeric 判断是否是正数
func isPositiveNumeric(value pgtype.Numeric) bool {
	rat, ok := numericRat(value)
	return ok && rat.Sign() > 0
}

// numericLessOrEqual 判断 left 是否小于等于 right
func numericLessOrEqual(left pgtype.Numeric, right pgtype.Numeric) bool {
	leftRat, leftOK := numericRat(left)
	rightRat, rightOK := numericRat(right)
	if !leftOK || !rightOK {
		return false
	}

	return leftRat.Cmp(rightRat) <= 0
}
