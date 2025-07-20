package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/Hobrus/gophermarket/internal/domain"
)

type stubOrderRepo struct {
	addFunc func(ctx context.Context, num string, userID int64, status string) (error, error, error)
}

func (s *stubOrderRepo) Add(ctx context.Context, num string, userID int64, status string) (error, error, error) {
	if s.addFunc != nil {
		return s.addFunc(ctx, num, userID, status)
	}
	return nil, nil, nil
}

func (s *stubOrderRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Order, error) {
	return nil, nil
}
func (s *stubOrderRepo) GetUnprocessed(ctx context.Context, limit int) ([]domain.Order, error) {
	return nil, nil
}
func (s *stubOrderRepo) UpdateStatus(ctx context.Context, num, status string, accrual *decimal.Decimal) error {
	return nil
}
func (s *stubOrderRepo) SumProcessedAccrualByUser(ctx context.Context, userID int64) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func TestOrderService_Add(t *testing.T) {
	repo := &stubOrderRepo{addFunc: func(ctx context.Context, num string, userID int64, status string) (error, error, error) {
		if num != "123" || userID != 1 || status != "NEW" {
			t.Fatalf("unexpected args %s %d %s", num, userID, status)
		}
		return nil, nil, nil
	}}
	svc := NewOrderService(repo)

	if errSelf, errOther, err := svc.Add(context.Background(), 1, "123"); errSelf != nil || errOther != nil || err != nil {
		t.Fatalf("unexpected return %v %v %v", errSelf, errOther, err)
	}
}

func TestOrderService_AddConflict(t *testing.T) {
	repo := &stubOrderRepo{addFunc: func(ctx context.Context, num string, userID int64, status string) (error, error, error) {
		return domain.ErrConflictSelf, nil, nil
	}}
	svc := NewOrderService(repo)

	errSelf, errOther, err := svc.Add(context.Background(), 1, "123")
	if !errors.Is(errSelf, domain.ErrConflictSelf) || errOther != nil || err != nil {
		t.Fatalf("unexpected errors %v %v %v", errSelf, errOther, err)
	}
}
