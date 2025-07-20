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

	"github.com/Hobrus/gophermarket/internal/domain"
	"github.com/Hobrus/gophermarket/internal/storage/postgres"
)

func setupPostgresBal(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	if os.Getenv("ENABLE_DOCKER_TESTS") == "" {
		t.Skip("skipping docker dependent tests; set ENABLE_DOCKER_TESTS=1 to run")
	}

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
	pool, teardown := setupPostgresBal(t)
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

type stubOrderRepoBal struct{ calls int }

func (s *stubOrderRepoBal) Add(ctx context.Context, num string, userID int64, status string) (error, error, error) {
	return nil, nil, nil
}
func (s *stubOrderRepoBal) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Order, error) {
	return nil, nil
}
func (s *stubOrderRepoBal) GetUnprocessed(ctx context.Context, limit int) ([]domain.Order, error) {
	return nil, nil
}
func (s *stubOrderRepoBal) UpdateStatus(ctx context.Context, num, status string, accrual *decimal.Decimal) error {
	return nil
}
func (s *stubOrderRepoBal) SumProcessedAccrualByUser(ctx context.Context, userID int64) (decimal.Decimal, error) {
	s.calls++
	return decimal.NewFromInt(10), nil
}

type stubWithdrawalRepoBal struct{ calls int }

func (s *stubWithdrawalRepoBal) Create(ctx context.Context, num string, userID int64, amount decimal.Decimal) error {
	return nil
}
func (s *stubWithdrawalRepoBal) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Withdrawal, error) {
	return nil, nil
}
func (s *stubWithdrawalRepoBal) SumByUser(ctx context.Context, userID int64) (decimal.Decimal, error) {
	s.calls++
	return decimal.NewFromInt(5), nil
}

func TestBalanceService_Cache(t *testing.T) {
	oRepo := &stubOrderRepoBal{}
	wRepo := &stubWithdrawalRepoBal{}
	svc := NewBalanceService(oRepo, wRepo)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		bal, err := svc.GetBalance(ctx, 1)
		if err != nil {
			t.Fatalf("get balance: %v", err)
		}
		if !bal.Current.Equal(decimal.NewFromInt(5)) || !bal.Withdrawn.Equal(decimal.NewFromInt(5)) {
			t.Fatalf("unexpected balance %+v", bal)
		}
	}
	if oRepo.calls != 1 || wRepo.calls != 1 {
		t.Fatalf("expected single repo call, got %d %d", oRepo.calls, wRepo.calls)
	}
}
