package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/Hobrus/gophermarket/internal/domain"
)

type stubBalanceSvc struct {
	getFunc func(ctx context.Context, userID int64) (domain.Balance, error)
}

func (s *stubBalanceSvc) GetBalance(ctx context.Context, userID int64) (domain.Balance, error) {
	return s.getFunc(ctx, userID)
}

func TestBalance_Unauthorized(t *testing.T) {
	svc := &stubBalanceSvc{}
	h := Balance(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

func TestBalance_Success(t *testing.T) {
	svc := &stubBalanceSvc{getFunc: func(ctx context.Context, userID int64) (domain.Balance, error) {
		if userID != 1 {
			t.Fatalf("unexpected user id %d", userID)
		}
		return domain.Balance{Current: decimal.NewFromFloat(10.5), Withdrawn: decimal.NewFromInt(3)}, nil
	}}
	h := Balance(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	ctx := context.WithValue(req.Context(), userIDKey, int64(1))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var resp struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Current != 10.5 || resp.Withdrawn != 3 {
		t.Fatalf("unexpected resp %+v", resp)
	}
}

func TestBalance_Zero(t *testing.T) {
	svc := &stubBalanceSvc{getFunc: func(ctx context.Context, userID int64) (domain.Balance, error) {
		return domain.Balance{Current: decimal.Zero, Withdrawn: decimal.Zero}, nil
	}}
	h := Balance(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(5)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var resp struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Current != 0 || resp.Withdrawn != 0 {
		t.Fatalf("unexpected resp %+v", resp)
	}
}

func TestBalance_Error(t *testing.T) {
	svc := &stubBalanceSvc{getFunc: func(ctx context.Context, userID int64) (domain.Balance, error) {
		return domain.Balance{}, errors.New("fail")
	}}
	h := Balance(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(1)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.StatusCode)
	}
}
