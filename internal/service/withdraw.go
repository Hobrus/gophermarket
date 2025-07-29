package service

import (
	"context"

	"github.com/Hobrus/gophermarket/internal/repository"
	"github.com/shopspring/decimal"
)

// WithdrawService provides withdrawal operations.
type WithdrawService struct {
	withdrawals repository.WithdrawalRepo
	inval       BalanceInvalidator
}

// NewWithdrawService creates a new WithdrawService instance.
func NewWithdrawService(w repository.WithdrawalRepo, b BalanceInvalidator) *WithdrawService {
	return &WithdrawService{withdrawals: w, inval: b}
}

// Withdraw deducts amount from user's balance if sufficient.
// Returns ErrInsufficientFunds if current balance is less than amount.
func (s *WithdrawService) Withdraw(ctx context.Context, userID int64, number string, amount decimal.Decimal) error {
	if err := s.withdrawals.Withdraw(ctx, number, userID, amount); err != nil {
		return err
	}
	if s.inval != nil {
		s.inval.Invalidate(userID)
	}
	return nil
}
