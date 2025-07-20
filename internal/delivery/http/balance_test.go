package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/Hobrus/gophermarket/internal/domain"
)

type stubBalance struct {
	withdrawFunc func(ctx context.Context, userID int64, order string, amount decimal.Decimal) error
}

func (s *stubBalance) Withdraw(ctx context.Context, userID int64, order string, amount decimal.Decimal) error {
	return s.withdrawFunc(ctx, userID, order, amount)
}

func TestWithdraw_Success(t *testing.T) {
	svc := &stubBalance{withdrawFunc: func(ctx context.Context, userID int64, order string, amount decimal.Decimal) error {
		if userID != 1 || order != "2377225624" || !amount.Equal(decimal.NewFromInt(751)) {
			t.Fatalf("unexpected args %d %s %s", userID, order, amount.String())
		}
		return nil
	}}
	router := NewBalanceRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(`{"order":"2377225624","sum":751}`))
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(1)))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestWithdraw_Insufficient(t *testing.T) {
	svc := &stubBalance{withdrawFunc: func(ctx context.Context, userID int64, order string, amount decimal.Decimal) error {
		return domain.ErrInsufficientBalance
	}}
	router := NewBalanceRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(`{"order":"2377225624","sum":751}`))
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(1)))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d", w.Result().StatusCode)
	}
}

func TestWithdraw_InvalidOrder(t *testing.T) {
	called := false
	svc := &stubBalance{withdrawFunc: func(ctx context.Context, userID int64, order string, amount decimal.Decimal) error {
		called = true
		return nil
	}}
	router := NewBalanceRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(`{"order":"123","sum":751}`))
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(1)))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Result().StatusCode)
	}
	if called {
		t.Fatal("service should not be called")
	}
}
