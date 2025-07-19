package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupDB(t *testing.T) (*DB, tc.Container) {
	t.Helper()
	ctx := context.Background()

	req := tc.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "pass",
			"POSTGRES_USER":     "user",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	cont, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}

	host, err := cont.Host(ctx)
	if err != nil {
		cont.Terminate(ctx)
		t.Fatal(err)
	}
	port, err := cont.MappedPort(ctx, "5432")
	if err != nil {
		cont.Terminate(ctx)
		t.Fatal(err)
	}

	dsn := fmt.Sprintf("postgres://user:pass@%s:%s/testdb?sslmode=disable", host, port.Port())
	store, err := New(ctx, dsn)
	if err != nil {
		cont.Terminate(ctx)
		t.Fatal(err)
	}

	if err := runMigrations(ctx, store); err != nil {
		cont.Terminate(ctx)
		t.Fatal(err)
	}

	return store, cont
}

func runMigrations(ctx context.Context, s *DB) error {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "0001_init.up.sql"))
	if err != nil {
		return err
	}
	statements := strings.Split(string(data), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := s.pool.Exec(ctx2, stmt)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

func TestOrderAddConflicts(t *testing.T) {
	db, cont := setupDB(t)
	defer cont.Terminate(context.Background())
	defer db.Close()

	ctx := context.Background()
	userRepo := db.User()
	orderRepo := db.Order()

	u1, err := userRepo.Create(ctx, "user1", "hash1")
	if err != nil {
		t.Fatalf("create user1: %v", err)
	}
	u2, err := userRepo.Create(ctx, "user2", "hash2")
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}

	if _, _, err := orderRepo.Add(ctx, "42", u1, "NEW"); err != nil {
		t.Fatalf("add order: %v", err)
	}
	if errSelf, errOther, err := orderRepo.Add(ctx, "42", u1, "NEW"); err != nil || errSelf == nil || errOther != nil {
		t.Fatalf("expected self conflict, got self=%v other=%v err=%v", errSelf, errOther, err)
	}
	if errSelf, errOther, err := orderRepo.Add(ctx, "42", u2, "NEW"); err != nil || errSelf != nil || errOther == nil {
		t.Fatalf("expected other conflict, got self=%v other=%v err=%v", errSelf, errOther, err)
	}
}

func TestSumByUser(t *testing.T) {
	db, cont := setupDB(t)
	defer cont.Terminate(context.Background())
	defer db.Close()

	ctx := context.Background()
	userRepo := db.User()
	wRepo := db.Withdrawal()

	uid, err := userRepo.Create(ctx, "user", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := wRepo.Create(ctx, "ord1", uid, decimal.NewFromInt(10)); err != nil {
		t.Fatalf("withdrawal: %v", err)
	}
	if err := wRepo.Create(ctx, "ord2", uid, decimal.NewFromInt(5)); err != nil {
		t.Fatalf("withdrawal: %v", err)
	}
	sum, err := wRepo.SumByUser(ctx, uid)
	if err != nil {
		t.Fatalf("sum: %v", err)
	}
	if !sum.Equal(decimal.NewFromInt(15)) {
		t.Fatalf("expected 15, got %s", sum)
	}
}
