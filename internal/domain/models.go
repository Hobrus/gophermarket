package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// User represents service user.
type User struct {
	ID           int64
	Login        string
	PasswordHash string
}

// Order represents user order uploaded for accrual processing.
type Order struct {
	Number     string
	UserID     int64
	Status     string
	Accrual    *decimal.Decimal
	UploadedAt time.Time
}

// Withdrawal represents loyalty points withdrawal by a user.
type Withdrawal struct {
	Number      string
	UserID      int64
	Amount      decimal.Decimal
	ProcessedAt time.Time
}
