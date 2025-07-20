package service

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/Hobrus/gophermarket/internal/repository"
)

// BalanceService provides user balance information.
type BalanceService struct {
	orders      repository.OrderRepo
	withdrawals repository.WithdrawalRepo
}

// NewBalanceService creates a new BalanceService.
func NewBalanceService(o repository.OrderRepo, w repository.WithdrawalRepo) *BalanceService {
	return &BalanceService{orders: o, withdrawals: w}
}

// GetBalance returns current balance and total withdrawn for user.
func (s *BalanceService) GetBalance(ctx context.Context, userID int64) (decimal.Decimal, decimal.Decimal, error) {
	accrual, err := s.orders.SumAccrualByUser(ctx, userID)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	withdrawn, err := s.withdrawals.SumByUser(ctx, userID)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	return accrual.Sub(withdrawn), withdrawn, nil
}
