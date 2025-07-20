package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Hobrus/gophermarket/internal/domain"
	"github.com/shopspring/decimal"
)

type stubWithdrawSvc struct {
	withdrawFunc func(ctx context.Context, userID int64, number string, amount decimal.Decimal) error
}

func (s *stubWithdrawSvc) Withdraw(ctx context.Context, userID int64, number string, amount decimal.Decimal) error {
	return s.withdrawFunc(ctx, userID, number, amount)
}

func TestWithdraw_Success(t *testing.T) {
	svc := &stubWithdrawSvc{withdrawFunc: func(ctx context.Context, userID int64, number string, amount decimal.Decimal) error {
		if userID != 1 || number != "2377225624" || !amount.Equal(decimal.NewFromInt(10)) {
			t.Fatalf("unexpected args")
		}
		return nil
	}}
	h := Withdraw(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(`{"order":"2377225624","sum":10}`))
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(1)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestWithdraw_Insufficient(t *testing.T) {
	svc := &stubWithdrawSvc{withdrawFunc: func(ctx context.Context, userID int64, number string, amount decimal.Decimal) error {
		return domain.ErrInsufficientFunds
	}}
	h := Withdraw(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(`{"order":"2377225624","sum":5}`))
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(2)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d", res.StatusCode)
	}
}

func TestWithdraw_InvalidOrder(t *testing.T) {
	svc := &stubWithdrawSvc{withdrawFunc: func(ctx context.Context, userID int64, number string, amount decimal.Decimal) error {
		t.Errorf("should not be called")
		return nil
	}}
	h := Withdraw(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(`{"order":"123","sum":1}`))
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(1)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", res.StatusCode)
	}
}
