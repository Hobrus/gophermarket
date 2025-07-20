package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Hobrus/gophermarket/internal/domain"
	"github.com/Hobrus/gophermarket/internal/repository"
)

const timeout = 5 * time.Second

// New creates repositories backed by pgx pool.
func New(pool *pgxpool.Pool) (repository.UserRepo, repository.OrderRepo, repository.WithdrawalRepo) {
	return &userRepo{pool}, &orderRepo{pool}, &withdrawalRepo{pool}
}

type userRepo struct{ pool *pgxpool.Pool }

type orderRepo struct{ pool *pgxpool.Pool }

type withdrawalRepo struct{ pool *pgxpool.Pool }

func beginTx(ctx context.Context, pool *pgxpool.Pool) (pgx.Tx, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	return tx, ctx, cancel, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// -- UserRepo implementation --

func (r *userRepo) Create(ctx context.Context, login, hash string) (int64, error) {
	tx, ctx, cancel, err := beginTx(ctx, r.pool)
	if err != nil {
		return 0, err
	}
	defer cancel()
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO users (login, password_hash) VALUES ($1,$2) RETURNING id`, login, hash).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, domain.ErrConflictSelf
		}
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *userRepo) GetByLogin(ctx context.Context, login string) (domain.User, error) {
	tx, ctx, cancel, err := beginTx(ctx, r.pool)
	if err != nil {
		return domain.User{}, err
	}
	defer cancel()
	defer tx.Rollback(ctx)

	var u domain.User
	err = tx.QueryRow(ctx, `SELECT id, login, password_hash FROM users WHERE login=$1`, login).
		Scan(&u.ID, &u.Login, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// -- OrderRepo implementation --

func (r *orderRepo) Add(ctx context.Context, num string, userID int64, status string) (error, error, error) {
	tx, ctx, cancel, err := beginTx(ctx, r.pool)
	if err != nil {
		return nil, nil, err
	}
	defer cancel()
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO orders (number, user_id, status) VALUES ($1,$2,$3)`, num, userID, status)
	if err != nil {
		if isUniqueViolation(err) {
			var existing int64
			err2 := tx.QueryRow(ctx, `SELECT user_id FROM orders WHERE number=$1`, num).Scan(&existing)
			if err2 != nil {
				return nil, nil, err2
			}
			if existing == userID {
				return domain.ErrConflictSelf, nil, nil
			}
			return nil, domain.ErrConflictOther, nil
		}
		return nil, nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return nil, nil, nil
}

func (r *orderRepo) ListByUser(ctx context.Context, userID int64) ([]domain.Order, error) {
	tx, ctx, cancel, err := beginTx(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	defer cancel()
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT number, user_id, status, accrual, uploaded_at FROM orders WHERE user_id=$1 ORDER BY uploaded_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		err = rows.Scan(&o.Number, &o.UserID, &o.Status, &o.Accrual, &o.UploadedAt)
		if err != nil {
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

func (r *orderRepo) GetUnprocessed(ctx context.Context, limit int) ([]domain.Order, error) {
	tx, ctx, cancel, err := beginTx(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	defer cancel()
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT number, user_id, status, accrual, uploaded_at FROM orders WHERE status IN ('NEW','PROCESSING') ORDER BY uploaded_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		err = rows.Scan(&o.Number, &o.UserID, &o.Status, &o.Accrual, &o.UploadedAt)
		if err != nil {
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

func (r *orderRepo) UpdateStatus(ctx context.Context, num, status string, accrual *decimal.Decimal) error {
	tx, ctx, cancel, err := beginTx(ctx, r.pool)
	if err != nil {
		return err
	}
	defer cancel()
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE orders SET status=$2, accrual=$3 WHERE number=$1`, num, status, accrual)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *orderRepo) SumProcessedAccrualByUser(ctx context.Context, userID int64) (decimal.Decimal, error) {
	tx, ctx, cancel, err := beginTx(ctx, r.pool)
	if err != nil {
		return decimal.Zero, err
	}
	defer cancel()
	defer tx.Rollback(ctx)

	var sum decimal.Decimal
	err = tx.QueryRow(ctx, `SELECT COALESCE(SUM(accrual),0) FROM orders WHERE status='PROCESSED' AND user_id=$1`, userID).Scan(&sum)
	if err != nil {
		return decimal.Zero, err
	}
	if err = tx.Commit(ctx); err != nil {
		return decimal.Zero, err
	}
	return sum, nil
}

// -- WithdrawalRepo implementation --

func (r *withdrawalRepo) Create(ctx context.Context, num string, userID int64, amount decimal.Decimal) error {
	tx, ctx, cancel, err := beginTx(ctx, r.pool)
	if err != nil {
		return err
	}
	defer cancel()
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO withdrawals (order_number, user_id, amount) VALUES ($1,$2,$3)`, num, userID, amount)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *withdrawalRepo) ListByUser(ctx context.Context, userID int64) ([]domain.Withdrawal, error) {
	tx, ctx, cancel, err := beginTx(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	defer cancel()
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT order_number, user_id, amount, processed_at FROM withdrawals WHERE user_id=$1 ORDER BY processed_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []domain.Withdrawal
	for rows.Next() {
		var w domain.Withdrawal
		if err = rows.Scan(&w.Number, &w.UserID, &w.Amount, &w.ProcessedAt); err != nil {
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

func (r *withdrawalRepo) SumByUser(ctx context.Context, userID int64) (decimal.Decimal, error) {
	tx, ctx, cancel, err := beginTx(ctx, r.pool)
	if err != nil {
		return decimal.Zero, err
	}
	defer cancel()
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
