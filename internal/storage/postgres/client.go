package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectWithRetry tries to create a pgx pool with the specified number of
// attempts. Between attempts it waits for the provided delay. The function
// returns the pool once Ping succeeds.
func ConnectWithRetry(ctx context.Context, cfg *pgxpool.Config, attempts int, delay time.Duration) (*pgxpool.Pool, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = pool.Ping(ctxPing)
			cancel()
			if err == nil {
				return pool, nil
			}
			pool.Close()
		}
		lastErr = err
		if i < attempts-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return nil, lastErr
}
