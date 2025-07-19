package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Hobrus/gophermarket/internal/domain"
)

// DB wraps pgxpool.Pool.
type DB struct {
	pool *pgxpool.Pool
}

// New connects to PostgreSQL and returns DB instance.
func New(ctx context.Context, dsn string) (*DB, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &DB{pool: pool}, nil
}

// Close closes underlying pool.
func (db *DB) Close() { db.pool.Close() }

// UserStorage implements repository.UserRepo.
type UserStorage struct{ db *DB }

func (db *DB) User() *UserStorage { return &UserStorage{db: db} }

func (s *UserStorage) Create(ctx context.Context, login, hash string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := s.db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO users(login, password_hash) VALUES($1,$2) RETURNING id`, login, hash).Scan(&id)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return 0, domain.ErrConflictSelf
		}
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *UserStorage) GetByLogin(ctx context.Context, login string) (domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := s.db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(ctx)

	var u domain.User
	err = tx.QueryRow(ctx, `SELECT id, login, password_hash FROM users WHERE login=$1`, login).Scan(&u.ID, &u.Login, &u.PasswordHash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// OrderStorage implements repository.OrderRepo.
type OrderStorage struct{ db *DB }

func (db *DB) Order() *OrderStorage { return &OrderStorage{db: db} }

func (s *OrderStorage) Add(ctx context.Context, num string, userID int64, status string) (errConflictSelf, errConflictOther, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := s.db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	_, execErr := tx.Exec(ctx, `INSERT INTO orders(number, user_id, status) VALUES($1,$2,$3)`, num, userID, status)
	if execErr != nil {
		if pgErr, ok := execErr.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			var existingUserID int64
			qErr := tx.QueryRow(ctx, `SELECT user_id FROM orders WHERE number=$1`, num).Scan(&existingUserID)
			if qErr != nil {
				return nil, nil, qErr
			}
			if existingUserID == userID {
				return domain.ErrConflictSelf, nil, nil
			}
			return nil, domain.ErrConflictOther, nil
		}
		return nil, nil, execErr
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return nil, nil, nil
}

func (s *OrderStorage) ListByUser(ctx context.Context, userID int64) ([]domain.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := s.db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT number, user_id, status, accrual, uploaded_at FROM orders WHERE user_id=$1 ORDER BY uploaded_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.Number, &o.UserID, &o.Status, &o.Accrual, &o.UploadedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *OrderStorage) GetUnprocessed(ctx context.Context, limit int) ([]domain.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := s.db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT number, user_id, status, accrual, uploaded_at FROM orders WHERE status IN ('NEW','PROCESSING') ORDER BY uploaded_at ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.Number, &o.UserID, &o.Status, &o.Accrual, &o.UploadedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return orders, nil
}

func (s *OrderStorage) UpdateStatus(ctx context.Context, num, status string, accrual *decimal.Decimal) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := s.db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmdTag, err := tx.Exec(ctx, `UPDATE orders SET status=$2, accrual=$3 WHERE number=$1`, num, status, accrual)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// WithdrawalStorage implements repository.WithdrawalRepo.
type WithdrawalStorage struct{ db *DB }

func (db *DB) Withdrawal() *WithdrawalStorage { return &WithdrawalStorage{db: db} }

func (s *WithdrawalStorage) Create(ctx context.Context, num string, userID int64, amount decimal.Decimal) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := s.db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO withdrawals(order_number,user_id,amount) VALUES($1,$2,$3)`, num, userID, amount)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *WithdrawalStorage) ListByUser(ctx context.Context, userID int64) ([]domain.Withdrawal, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := s.db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT order_number, user_id, amount, processed_at FROM withdrawals WHERE user_id=$1 ORDER BY processed_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []domain.Withdrawal
	for rows.Next() {
		var w domain.Withdrawal
		if err := rows.Scan(&w.Number, &w.UserID, &w.Amount, &w.ProcessedAt); err != nil {
			return nil, err
		}
		res = append(res, w)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *WithdrawalStorage) SumByUser(ctx context.Context, userID int64) (decimal.Decimal, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := s.db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return decimal.Zero, err
	}
	defer tx.Rollback(ctx)

	var sum decimal.Decimal
	err = tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0) FROM withdrawals WHERE user_id=$1`, userID).Scan(&sum)
	if err != nil {
		return decimal.Zero, err
	}
	if err = tx.Commit(ctx); err != nil {
		return decimal.Zero, err
	}
	return sum, nil
}

// Helper to execute migration statements separated by semicolon.
func ExecMigration(ctx context.Context, pool *pgxpool.Pool, sql string) error {
	stmts := strings.Split(sql, ";")
	for _, s := range stmts {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "--") {
			continue
		}
		if _, err := pool.Exec(ctx, s); err != nil {
			return err
		}
	}
	return nil
}
