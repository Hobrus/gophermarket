package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Hobrus/gophermarket/internal/domain"
)

type stubOrders struct {
	listFunc func(ctx context.Context, userID int64, limit, offset int) ([]domain.Order, error)
}

func (s *stubOrders) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Order, error) {
	return s.listFunc(ctx, userID, limit, offset)
}

func TestListOrders_NoOrders(t *testing.T) {
	svc := &stubOrders{listFunc: func(ctx context.Context, userID int64, limit, offset int) ([]domain.Order, error) {
		if limit != 50 || offset != 0 {
			t.Fatalf("unexpected pagination %d %d", limit, offset)
		}
		return []domain.Order{}, nil
	}}
	router := NewOrdersRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(1)))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

func TestListOrders_Sorted(t *testing.T) {
	t1 := time.Now().Add(-time.Hour)
	t2 := time.Now()
	accr := decimal.NewFromInt(5)
	orders := []domain.Order{
		{Number: "old", Status: "PROCESSED", Accrual: &accr, UploadedAt: t1},
		{Number: "new", Status: "NEW", UploadedAt: t2},
	}
	svc := &stubOrders{listFunc: func(ctx context.Context, userID int64, limit, offset int) ([]domain.Order, error) {
		if limit != 50 || offset != 0 {
			t.Fatalf("unexpected pagination %d %d", limit, offset)
		}
		return orders, nil
	}}
	router := NewOrdersRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(1)))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var resp []struct {
		Number     string   `json:"number"`
		Status     string   `json:"status"`
		Accrual    *float64 `json:"accrual,omitempty"`
		UploadedAt string   `json:"uploaded_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp))
	}
	if resp[0].Number != "new" {
		t.Fatalf("expected new first, got %s", resp[0].Number)
	}
	if resp[0].Accrual != nil {
		t.Fatal("unexpected accrual for new order")
	}
	if resp[1].Number != "old" {
		t.Fatalf("expected old second, got %s", resp[1].Number)
	}
	if resp[1].Accrual == nil || *resp[1].Accrual != 5 {
		t.Fatalf("unexpected accrual %v", resp[1].Accrual)
	}
}

func TestListOrders_Paging(t *testing.T) {
	svc := &stubOrders{listFunc: func(ctx context.Context, userID int64, limit, offset int) ([]domain.Order, error) {
		if limit != 1 || offset != 1 {
			t.Fatalf("unexpected pagination %d %d", limit, offset)
		}
		return []domain.Order{{Number: "one"}, {Number: "two"}, {Number: "three"}}[offset : offset+limit], nil
	}}
	router := NewOrdersRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders?limit=1&offset=1", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(5)))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var resp []struct {
		Number string `json:"number"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 1 || resp[0].Number != "two" {
		t.Fatalf("unexpected response %+v", resp)
	}
}
