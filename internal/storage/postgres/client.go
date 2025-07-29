package postgres

import (
	"context"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
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

// NewPool parses connection string, configures OpenTelemetry instrumentation,
// applies migrations and returns a ready-to-use pgx pool. It also retries
// connection attempts a few times before failing.
func NewPool(ctx context.Context, dsn string, tp trace.TracerProvider, mp metric.MeterProvider) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithTracerProvider(tp),
		otelpgx.WithMeterProvider(mp),
	)

	pool, err := ConnectWithRetry(ctx, cfg, 5, time.Second)
	if err != nil {
		return nil, err
	}

	if err = otelpgx.RecordStats(pool, otelpgx.WithStatsMeterProvider(mp)); err != nil {
		pool.Close()
		return nil, err
	}

	if err = ApplyMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
