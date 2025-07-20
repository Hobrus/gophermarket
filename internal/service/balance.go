package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"

	"github.com/Hobrus/gophermarket/internal/domain"
	"github.com/Hobrus/gophermarket/internal/repository"
)

// BalanceInvalidator allows invalidating cached balances.
type BalanceInvalidator interface {
	Invalidate(userID int64)
}

// BalanceService provides current balance calculation logic.
type BalanceService struct {
	orders      repository.OrderRepo
	withdrawals repository.WithdrawalRepo

	mu    sync.Mutex
	cache *lru.Cache
	ttl   time.Duration
}

// NewBalanceService creates a new BalanceService instance.
func NewBalanceService(o repository.OrderRepo, w repository.WithdrawalRepo) *BalanceService {
	return &BalanceService{
		orders:      o,
		withdrawals: w,
		cache:       lru.New(0),
		ttl:         30 * time.Second,
	}
}

// GetBalance returns current and withdrawn amounts for user.
func (s *BalanceService) GetBalance(ctx context.Context, userID int64) (domain.Balance, error) {
	key := fmt.Sprintf("balance:%d", userID)

	s.mu.Lock()
	if v, ok := s.cache.Get(key); ok {
		item := v.(cacheItem)
		if time.Now().Before(item.exp) {
			bal := item.bal
			s.mu.Unlock()
			return bal, nil
		}
		s.cache.Remove(key)
	}
	s.mu.Unlock()

	totalAccrual, err := s.orders.SumProcessedAccrualByUser(ctx, userID)
	if err != nil {
		return domain.Balance{}, err
	}
	totalWithdrawn, err := s.withdrawals.SumByUser(ctx, userID)
	if err != nil {
		return domain.Balance{}, err
	}
	current := totalAccrual.Sub(totalWithdrawn)
	bal := domain.Balance{Current: current, Withdrawn: totalWithdrawn}

	s.mu.Lock()
	s.cache.Add(key, cacheItem{bal: bal, exp: time.Now().Add(s.ttl)})
	s.mu.Unlock()

	return bal, nil
}

// Invalidate removes cached balance for user if present.
func (s *BalanceService) Invalidate(userID int64) {
	key := fmt.Sprintf("balance:%d", userID)
	s.mu.Lock()
	s.cache.Remove(key)
	s.mu.Unlock()
}

type cacheItem struct {
	bal domain.Balance
	exp time.Time
}
