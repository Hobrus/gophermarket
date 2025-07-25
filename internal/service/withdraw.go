package service

import (
	"context"

	"github.com/Hobrus/gophermarket/internal/domain"
	"github.com/Hobrus/gophermarket/internal/repository"
	"github.com/shopspring/decimal"
)

// WithdrawService provides withdrawal operations.
type WithdrawService struct {
	orders      repository.OrderRepo
	withdrawals repository.WithdrawalRepo
	inval       BalanceInvalidator
}

// NewWithdrawService creates a new WithdrawService instance.
func NewWithdrawService(o repository.OrderRepo, w repository.WithdrawalRepo, b BalanceInvalidator) *WithdrawService {
	return &WithdrawService{orders: o, withdrawals: w, inval: b}
}

// Withdraw deducts amount from user's balance if sufficient.
// Returns ErrInsufficientFunds if current balance is less than amount.
func (s *WithdrawService) Withdraw(ctx context.Context, userID int64, number string, amount decimal.Decimal) error {
	totalAccrual, err := s.orders.SumProcessedAccrualByUser(ctx, userID)
	if err != nil {
		return err
	}
	totalWithdrawn, err := s.withdrawals.SumByUser(ctx, userID)
	if err != nil {
		return err
	}
	current := totalAccrual.Sub(totalWithdrawn)
	if current.Cmp(amount) < 0 {
		return domain.ErrInsufficientFunds
	}
	if err := s.withdrawals.Create(ctx, number, userID, amount); err != nil {
		return err
	}
	if s.inval != nil {
		s.inval.Invalidate(userID)
	}
	return nil
}
