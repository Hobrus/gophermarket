package service

import (
	"context"

	"github.com/Hobrus/gophermarket/internal/domain"
	"github.com/Hobrus/gophermarket/internal/repository"
)

// BalanceService provides current balance calculation logic.
type BalanceService struct {
	orders      repository.OrderRepo
	withdrawals repository.WithdrawalRepo
}

// NewBalanceService creates a new BalanceService instance.
func NewBalanceService(o repository.OrderRepo, w repository.WithdrawalRepo) *BalanceService {
	return &BalanceService{orders: o, withdrawals: w}
}

// GetBalance returns current and withdrawn amounts for user.
func (s *BalanceService) GetBalance(ctx context.Context, userID int64) (domain.Balance, error) {
	totalAccrual, err := s.orders.SumProcessedAccrualByUser(ctx, userID)
	if err != nil {
		return domain.Balance{}, err
	}
	totalWithdrawn, err := s.withdrawals.SumByUser(ctx, userID)
	if err != nil {
		return domain.Balance{}, err
	}
	current := totalAccrual.Sub(totalWithdrawn)
	return domain.Balance{Current: current, Withdrawn: totalWithdrawn}, nil
}
