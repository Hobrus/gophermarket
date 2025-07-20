package repository

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/Hobrus/gophermarket/internal/domain"
)

// UserRepo accesses user storage.
type UserRepo interface {
	// Create stores new user and returns its id.
	// Returns ErrConflictSelf on unique violation.
	Create(ctx context.Context, login, hash string) (int64, error)
	// GetByLogin returns user by login. Returns ErrNotFound if absent.
	GetByLogin(ctx context.Context, login string) (domain.User, error)
}

// OrderRepo accesses order storage.
type OrderRepo interface {
	// Add stores a new order for user.
	// Returns ErrConflictSelf if the order already belongs to this user,
	// ErrConflictOther if it belongs to another user.
	Add(ctx context.Context, num string, userID int64, status string) (errConflictSelf, errConflictOther, err error)
	// ListByUser returns orders uploaded by the user sorted by upload time desc.
	ListByUser(ctx context.Context, userID int64) ([]domain.Order, error)
	// GetUnprocessed returns a list of orders with status NEW or PROCESSING up to limit.
	GetUnprocessed(ctx context.Context, limit int) ([]domain.Order, error)
	// UpdateStatus updates order status and optional accrual.
	UpdateStatus(ctx context.Context, num, status string, accrual *decimal.Decimal) error
	// SumProcessedAccrualByUser returns total accrual for processed orders of the user.
	SumProcessedAccrualByUser(ctx context.Context, userID int64) (decimal.Decimal, error)
}

// WithdrawalRepo accesses withdrawals storage.
type WithdrawalRepo interface {
	// Create registers a withdrawal request for user.
	Create(ctx context.Context, num string, userID int64, amount decimal.Decimal) error
	// ListByUser returns withdrawal history for user sorted by processed time desc.
	ListByUser(ctx context.Context, userID int64) ([]domain.Withdrawal, error)
	// SumByUser returns total amount withdrawn by user.
	SumByUser(ctx context.Context, userID int64) (decimal.Decimal, error)
}
