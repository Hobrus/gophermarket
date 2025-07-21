package postgres

import (
	"context"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

// applyMigrations executes initial SQL schema.
// ApplyMigrations executes the initial SQL schema.
// It is safe to call multiple times because the queries
// in the migration script use "IF NOT EXISTS" clauses.
func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	path := filepath.Join("migrations", "0001_init.up.sql")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, string(b))
	return err
}
