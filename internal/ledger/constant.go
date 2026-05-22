package ledger

// ReservationStatus 表示余额预授权记录状态。
type ReservationStatus string

const (
	ReservationStatusAuthorized ReservationStatus = "authorized"
	ReservationStatusCaptured   ReservationStatus = "captured"
	ReservationStatusReleased   ReservationStatus = "released"
)

// EntryType 表示账本流水类型。
type EntryType string

const (
	EntryTypeCredit           EntryType = "credit"
	EntryTypeDebit            EntryType = "debit"
	EntryTypeRefund           EntryType = "refund"
	EntryTypeAdjustmentCredit EntryType = "adjustment_credit"
	EntryTypeAdjustmentDebit  EntryType = "adjustment_debit"
)
