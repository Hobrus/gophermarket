package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

type stubBalanceSvc struct {
	getBalanceFunc func(ctx context.Context, userID int64) (decimal.Decimal, decimal.Decimal, error)
}

func (s *stubBalanceSvc) GetBalance(ctx context.Context, userID int64) (decimal.Decimal, decimal.Decimal, error) {
	return s.getBalanceFunc(ctx, userID)
}

func TestBalance(t *testing.T) {
	svc := &stubBalanceSvc{getBalanceFunc: func(ctx context.Context, userID int64) (decimal.Decimal, decimal.Decimal, error) {
		if userID != 1 {
			t.Fatalf("unexpected userID %d", userID)
		}
		return decimal.NewFromInt(7), decimal.NewFromInt(3), nil
	}}
	h := balance(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(1)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}
	var resp struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Current != 7 || resp.Withdrawn != 3 {
		t.Fatalf("unexpected response %+v", resp)
	}
}

func TestBalance_Zero(t *testing.T) {
	svc := &stubBalanceSvc{getBalanceFunc: func(ctx context.Context, userID int64) (decimal.Decimal, decimal.Decimal, error) {
		return decimal.Zero, decimal.Zero, nil
	}}
	h := balance(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(5)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}
	var resp struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Current != 0 || resp.Withdrawn != 0 {
		t.Fatalf("unexpected response %+v", resp)
	}
}
