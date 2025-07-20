package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Hobrus/gophermarket/internal/domain"
)

type stubWithdrawalRepo struct {
	listFunc func(ctx context.Context, userID int64) ([]domain.Withdrawal, error)
}

func (s *stubWithdrawalRepo) ListByUser(ctx context.Context, userID int64) ([]domain.Withdrawal, error) {
	return s.listFunc(ctx, userID)
}

func TestWithdrawals_Unauthorized(t *testing.T) {
	repo := &stubWithdrawalRepo{}
	h := Withdrawals(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Result().StatusCode)
	}
}

func TestWithdrawals_NoContent(t *testing.T) {
	repo := &stubWithdrawalRepo{listFunc: func(ctx context.Context, userID int64) ([]domain.Withdrawal, error) {
		if userID != 7 {
			t.Fatalf("unexpected user id %d", userID)
		}
		return nil, nil
	}}
	h := Withdrawals(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	ctx := context.WithValue(req.Context(), userIDKey, int64(7))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Result().StatusCode)
	}
}

func TestWithdrawals_Success(t *testing.T) {
	ts1 := time.Date(2020, 12, 9, 16, 9, 57, 0, time.FixedZone("", 3*3600))
	ts2 := ts1.Add(-time.Hour)
	repo := &stubWithdrawalRepo{listFunc: func(ctx context.Context, userID int64) ([]domain.Withdrawal, error) {
		if userID != 1 {
			t.Fatalf("unexpected user id %d", userID)
		}
		return []domain.Withdrawal{
			{Number: "1", Amount: decimal.NewFromInt(5), ProcessedAt: ts1},
			{Number: "2", Amount: decimal.NewFromFloat(7.5), ProcessedAt: ts2},
		}, nil
	}}
	h := Withdrawals(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	ctx := context.WithValue(req.Context(), userIDKey, int64(1))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}
	var res []struct {
		Order       string  `json:"order"`
		Sum         float64 `json:"sum"`
		ProcessedAt string  `json:"processed_at"`
	}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 items, got %d", len(res))
	}
	if res[0].Order != "1" || res[0].Sum != 5 || res[0].ProcessedAt != ts1.Format(time.RFC3339) {
		t.Fatal("first item mismatch")
	}
	if res[1].Order != "2" || res[1].Sum != 7.5 || res[1].ProcessedAt != ts2.Format(time.RFC3339) {
		t.Fatal("second item mismatch")
	}
}

func TestWithdrawals_Error(t *testing.T) {
	repo := &stubWithdrawalRepo{listFunc: func(ctx context.Context, userID int64) ([]domain.Withdrawal, error) {
		return nil, errors.New("fail")
	}}
	h := Withdrawals(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
	ctx := context.WithValue(req.Context(), userIDKey, int64(1))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Result().StatusCode)
	}
}
