package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Hobrus/gophermarket/internal/storage/postgres"
)

func setupPostgres(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image: "postgres:15-alpine",
		Env: map[string]string{
			"POSTGRES_PASSWORD": "pass",
			"POSTGRES_USER":     "user",
			"POSTGRES_DB":       "test",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}

	dsn := fmt.Sprintf("postgres://user:pass@%s:%s/test?sslmode=disable", host, port.Port())
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(ctxPing); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join("migrations", "0001_init.up.sql")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(b)); err != nil {
		t.Fatal(err)
	}

	return pool, func() {
		pool.Close()
		container.Terminate(context.Background())
	}
}

func TestBalanceService_GetBalance(t *testing.T) {
	pool, teardown := setupPostgres(t)
	defer teardown()

	userRepo, orderRepo, withdrawalRepo := postgres.New(pool)
	ctx := context.Background()

	uid, err := userRepo.Create(ctx, "login", "hash")
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := orderRepo.Add(ctx, "o1", uid, "NEW"); err != nil {
		t.Fatalf("add order: %v", err)
	}
	accrual := decimal.NewFromInt(10)
	if err := orderRepo.UpdateStatus(ctx, "o1", "PROCESSED", &accrual); err != nil {
		t.Fatalf("update status: %v", err)
	}

	if _, _, err := orderRepo.Add(ctx, "o2", uid, "NEW"); err != nil {
		t.Fatalf("add order: %v", err)
	}
	accrual2 := decimal.NewFromInt(5)
	if err := orderRepo.UpdateStatus(ctx, "o2", "PROCESSED", &accrual2); err != nil {
		t.Fatalf("update status: %v", err)
	}

	if _, _, err := orderRepo.Add(ctx, "o3", uid, "NEW"); err != nil {
		t.Fatalf("add order: %v", err)
	}

	if err := withdrawalRepo.Create(ctx, "w1", uid, decimal.NewFromInt(8)); err != nil {
		t.Fatalf("withdraw create: %v", err)
	}

	svc := NewBalanceService(orderRepo, withdrawalRepo)
	bal, err := svc.GetBalance(ctx, uid)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}

	if !bal.Current.Equal(decimal.NewFromInt(7)) {
		t.Errorf("expected current 7, got %s", bal.Current)
	}
	if !bal.Withdrawn.Equal(decimal.NewFromInt(8)) {
		t.Errorf("expected withdrawn 8, got %s", bal.Withdrawn)
	}
}
