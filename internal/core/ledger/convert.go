package ledger

import "github.com/jackc/pgx/v5/pgtype"

// pgtypeInt8ToInt64Ptr 将 pgtype.Int8 转成可选 int64 指针。
func pgtypeInt8ToInt64Ptr(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}

	return &value.Int64
}

// int64PtrToPgtypeInt8 将可选 int64 指针转成 pgtype.Int8。
func int64PtrToPgtypeInt8(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{Valid: false}
	}

	return pgtype.Int8{Int64: *value, Valid: true}
}

// sameOptionalInt64 比较数据库可空 int8 和领域层可选 int64 是否相同。
func sameOptionalInt64(left pgtype.Int8, right *int64) bool {
	if !left.Valid {
		return right == nil
	}
	if right == nil {
		return false
	}

	return left.Int64 == *right
}
