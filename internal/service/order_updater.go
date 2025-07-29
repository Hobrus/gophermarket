package service

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Hobrus/gophermarket/internal/accrualclient"
	"github.com/Hobrus/gophermarket/internal/repository"
)

// OrderUpdater periodically updates order statuses using external accrual service.
type OrderUpdater struct {
	repo       repository.OrderRepo
	client     accrualclient.Client
	inval      BalanceInvalidator
	sleepUntil int64
}

func (u *OrderUpdater) wait(ctx context.Context) error {
	until := time.Unix(0, atomic.LoadInt64(&u.sleepUntil))
	d := time.Until(until)
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
	}
	return nil
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
			if err := u.wait(ctx); err != nil {
				wg.Wait()
				return
			}
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
					if err := u.wait(ctx); err != nil {
						return
					}

					status, accrual, retry, err := u.client.Get(ctx, num)
					if err != nil {
						return
					}
					if retry > 0 {
						until := time.Now().Add(retry).UnixNano()
						for {
							old := atomic.LoadInt64(&u.sleepUntil)
							if until <= old {
								break
							}
							if atomic.CompareAndSwapInt64(&u.sleepUntil, old, until) {
								break
							}
						}
						return
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
