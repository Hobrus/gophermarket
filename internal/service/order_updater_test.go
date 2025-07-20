package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Hobrus/gophermarket/internal/accrualclient"
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

func TestOrderUpdater_Run(t *testing.T) {
	pool, teardown := setupPostgres(t)
	defer teardown()

	userRepo, orderRepo, _ := postgres.New(pool)

	ctx := context.Background()
	uid, err := userRepo.Create(ctx, "user", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := orderRepo.Add(ctx, "o1", uid, "NEW"); err != nil {
		t.Fatalf("add order: %v", err)
	}
	if _, _, err := orderRepo.Add(ctx, "o2", uid, "NEW"); err != nil {
		t.Fatalf("add order: %v", err)
	}

	var mu sync.Mutex
	calls := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		num := path.Base(r.URL.Path)
		mu.Lock()
		c := calls[num]
		calls[num] = c + 1
		mu.Unlock()

		switch num {
		case "o1":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"order": num, "status": "PROCESSED", "accrual": 10})
		case "o2":
			if c == 0 {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"order": num, "status": "PROCESSED", "accrual": 5})
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	upd := NewOrderUpdater(orderRepo, accrualclient.New(srv.URL), nil)
	runCtx, cancel := context.WithCancel(context.Background())
	go upd.Run(runCtx, 2, 2, 200*time.Millisecond)

	time.Sleep(2 * time.Second)
	cancel()
	time.Sleep(200 * time.Millisecond)

	orders, err := orderRepo.ListByUser(ctx, uid, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(orders))
	}
	for _, o := range orders {
		switch o.Number {
		case "o1":
			if o.Status != "PROCESSED" || o.Accrual == nil || !o.Accrual.Equal(decimal.NewFromInt(10)) {
				t.Fatalf("o1 not processed: %+v", o)
			}
		case "o2":
			if o.Status != "PROCESSED" || o.Accrual == nil || !o.Accrual.Equal(decimal.NewFromInt(5)) {
				t.Fatalf("o2 not processed: %+v", o)
			}
		}
	}
}
