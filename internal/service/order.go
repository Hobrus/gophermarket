package service

import (
	"context"

	"github.com/Hobrus/gophermarket/internal/repository"
)

// OrderService provides order-related operations.
type OrderService struct {
	repo repository.OrderRepo
}

// NewOrderService creates a new OrderService instance.
func NewOrderService(repo repository.OrderRepo) *OrderService {
	return &OrderService{repo: repo}
}

// Add registers a new order with status NEW.
func (s *OrderService) Add(ctx context.Context, userID int64, number string) (errConflictSelf, errConflictOther, err error) {
	return s.repo.Add(ctx, number, userID, "NEW")
}
