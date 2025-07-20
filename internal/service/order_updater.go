package service

import (
	"context"
	"sync"
	"time"

	"github.com/Hobrus/gophermarket/internal/accrualclient"
	"github.com/Hobrus/gophermarket/internal/repository"
)

// OrderUpdater periodically updates order statuses using external accrual service.
type OrderUpdater struct {
	repo   repository.OrderRepo
	client accrualclient.Client
	inval  BalanceInvalidator
}

// NewOrderUpdater creates a new updater instance.
func NewOrderUpdater(r repository.OrderRepo, c accrualclient.Client, b BalanceInvalidator) *OrderUpdater {
	return &OrderUpdater{repo: r, client: c, inval: b}
}

// Run starts background workers that update orders until ctx is done.
func (u *OrderUpdater) Run(ctx context.Context, parallel, batch int, interval time.Duration) {
	sem := make(chan struct{}, parallel)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var wg sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-ticker.C:
			orders, err := u.repo.GetUnprocessed(ctx, batch)
			if err != nil {
				continue
			}
		ordersLoop:
			for _, o := range orders {
				uid := o.UserID
				select {
				case <-ctx.Done():
					break ordersLoop
				case sem <- struct{}{}:
				}
				wg.Add(1)
				go func(num string, uid int64) {
					defer func() {
						<-sem
						wg.Done()
					}()

					status, accrual, retry, err := u.client.Get(ctx, num)
					if err != nil {
						return
					}
					if retry > 0 {
						t := time.NewTimer(retry)
						select {
						case <-ctx.Done():
							t.Stop()
							return
						case <-t.C:
						}
					}
					if status == "" {
						return
					}
					_ = u.repo.UpdateStatus(ctx, num, status, accrual)
					if status == "PROCESSED" && u.inval != nil {
						u.inval.Invalidate(uid)
					}
				}(o.Number, uid)
			}
		}
	}
}
