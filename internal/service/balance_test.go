package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/Hobrus/gophermarket/internal/domain"
)

type stubOrderRepo struct {
	sumFunc func(ctx context.Context, userID int64) (decimal.Decimal, error)
}

func (s *stubOrderRepo) Add(ctx context.Context, num string, userID int64, status string) (error, error, error) {
	return nil, nil, nil
}
func (s *stubOrderRepo) ListByUser(ctx context.Context, userID int64) ([]domain.Order, error) {
	return nil, nil
}
func (s *stubOrderRepo) GetUnprocessed(ctx context.Context, limit int) ([]domain.Order, error) {
	return nil, nil
}
func (s *stubOrderRepo) UpdateStatus(ctx context.Context, num, status string, accrual *decimal.Decimal) error {
	return nil
}
func (s *stubOrderRepo) SumAccrualByUser(ctx context.Context, userID int64) (decimal.Decimal, error) {
	return s.sumFunc(ctx, userID)
}

type stubWithdrawalRepo struct {
	sumFunc func(ctx context.Context, userID int64) (decimal.Decimal, error)
}

func (s *stubWithdrawalRepo) Create(ctx context.Context, num string, userID int64, amount decimal.Decimal) error {
	return nil
}
func (s *stubWithdrawalRepo) ListByUser(ctx context.Context, userID int64) ([]domain.Withdrawal, error) {
	return nil, nil
}
func (s *stubWithdrawalRepo) SumByUser(ctx context.Context, userID int64) (decimal.Decimal, error) {
	return s.sumFunc(ctx, userID)
}

func TestBalanceService_GetBalance(t *testing.T) {
	or := &stubOrderRepo{sumFunc: func(ctx context.Context, userID int64) (decimal.Decimal, error) {
		if userID != 1 {
			t.Fatalf("unexpected userID %d", userID)
		}
		return decimal.NewFromInt(10), nil
	}}
	wr := &stubWithdrawalRepo{sumFunc: func(ctx context.Context, userID int64) (decimal.Decimal, error) {
		if userID != 1 {
			t.Fatalf("unexpected userID %d", userID)
		}
		return decimal.NewFromInt(3), nil
	}}
	svc := NewBalanceService(or, wr)

	current, withdrawn, err := svc.GetBalance(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if !current.Equal(decimal.NewFromInt(7)) {
		t.Errorf("unexpected current %s", current)
	}
	if !withdrawn.Equal(decimal.NewFromInt(3)) {
		t.Errorf("unexpected withdrawn %s", withdrawn)
	}
}

func TestBalanceService_Zero(t *testing.T) {
	svc := NewBalanceService(&stubOrderRepo{sumFunc: func(ctx context.Context, userID int64) (decimal.Decimal, error) {
		return decimal.Zero, nil
	}}, &stubWithdrawalRepo{sumFunc: func(ctx context.Context, userID int64) (decimal.Decimal, error) {
		return decimal.Zero, nil
	}})

	current, withdrawn, err := svc.GetBalance(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if !current.Equal(decimal.Zero) || !withdrawn.Equal(decimal.Zero) {
		t.Fatalf("expected zeros, got %s %s", current, withdrawn)
	}
}
