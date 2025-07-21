package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Hobrus/gophermarket/internal/domain"
)

func setupPostgres(t *testing.T) (*pgxpool.Pool, func()) {
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

	if err := ApplyMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}

	return pool, func() {
		pool.Close()
		container.Terminate(context.Background())
	}
}

func TestRepositories(t *testing.T) {
	pool, teardown := setupPostgres(t)
	defer teardown()

	userRepo, orderRepo, withdrawalRepo := New(pool)

	ctx := context.Background()

	// create user
	uid, err := userRepo.Create(ctx, "login", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err = userRepo.Create(ctx, "login", "hash"); !errors.Is(err, domain.ErrConflictSelf) {
		t.Fatalf("expected conflict")
	}

	u, err := userRepo.GetByLogin(ctx, "login")
	if err != nil || u.ID != uid {
		t.Fatalf("get user: %v", err)
	}

	// orders
	if errSelf, errOther, err := orderRepo.Add(ctx, "42", uid, "NEW"); err != nil || errSelf != nil || errOther != nil {
		t.Fatalf("add order: %v %v %v", errSelf, errOther, err)
	}

	if errSelf, errOther, err := orderRepo.Add(ctx, "42", uid, "NEW"); !errors.Is(errSelf, domain.ErrConflictSelf) || errOther != nil || err != nil {
		t.Fatalf("expected self conflict")
	}

	if _, _, err := orderRepo.Add(ctx, "43", uid, "NEW"); err != nil {
		t.Fatalf("add second order: %v", err)
	}

	uid2, err := userRepo.Create(ctx, "other", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if errSelf, errOther, err := orderRepo.Add(ctx, "42", uid2, "NEW"); !errors.Is(errOther, domain.ErrConflictOther) || errSelf != nil || err != nil {
		t.Fatalf("expected other conflict")
	}

	orders, err := orderRepo.ListByUser(ctx, uid, 50, 0)
	if err != nil || len(orders) != 2 {
		t.Fatalf("list orders: %v", err)
	}
	page, err := orderRepo.ListByUser(ctx, uid, 1, 1)
	if err != nil || len(page) != 1 || page[0].Number != "43" {
		t.Fatalf("paged orders: %v %v", err, page)
	}

	up, err := orderRepo.GetUnprocessed(ctx, 10)
	if err != nil || len(up) != 1 {
		t.Fatalf("get unprocessed: %v", err)
	}

	accrual := decimal.NewFromInt(10)
	if err := orderRepo.UpdateStatus(ctx, "42", "PROCESSED", &accrual); err != nil {
		t.Fatalf("update status: %v", err)
	}

	up, err = orderRepo.GetUnprocessed(ctx, 10)
	if err != nil || len(up) != 0 {
		t.Fatalf("get unprocessed after update: %v", err)
	}

	// withdrawals
	if err := withdrawalRepo.Create(ctx, "w1", uid, decimal.NewFromInt(5)); err != nil {
		t.Fatalf("withdraw create: %v", err)
	}
	if err := withdrawalRepo.Create(ctx, "w2", uid, decimal.NewFromInt(3)); err != nil {
		t.Fatalf("withdraw create: %v", err)
	}

	ws, err := withdrawalRepo.ListByUser(ctx, uid, 50, 0)
	if err != nil || len(ws) != 2 {
		t.Fatalf("list withdrawals: %v", err)
	}

	wpage, err := withdrawalRepo.ListByUser(ctx, uid, 1, 1)
	if err != nil || len(wpage) != 1 || wpage[0].Number != "w2" {
		t.Fatalf("paged withdrawals: %v %v", err, wpage)
	}

	sum, err := withdrawalRepo.SumByUser(ctx, uid)
	if err != nil || !sum.Equal(decimal.NewFromInt(8)) {
		t.Fatalf("sum withdrawals: %v %s", err, sum)
	}
}
