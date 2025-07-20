package service

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/Hobrus/gophermarket/internal/repository"
)

// BalanceService provides user balance operations.
type BalanceService struct {
	withdrawalRepo repository.WithdrawalRepo
}

// NewBalanceService creates BalanceService instance.
func NewBalanceService(w repository.WithdrawalRepo) *BalanceService {
	return &BalanceService{withdrawalRepo: w}
}

// Withdraw withdraws loyalty points for user.
func (s *BalanceService) Withdraw(ctx context.Context, userID int64, order string, amount decimal.Decimal) error {
	return s.withdrawalRepo.Withdraw(ctx, order, userID, amount)
}
