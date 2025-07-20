package http

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Hobrus/gophermarket/internal/domain"
)

// ListService defines method required for listing user orders.
type ListService interface {
	ListByUser(ctx context.Context, userID int64) ([]domain.Order, error)
}

// NewOrdersRouter creates chi router with user orders endpoints.
func NewOrdersRouter(svc ListService) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/user/orders", listOrders(svc))
	return r
}

func listOrders(svc ListService) http.HandlerFunc {
	type orderDTO struct {
		Number     string   `json:"number"`
		Status     string   `json:"status"`
		Accrual    *float64 `json:"accrual,omitempty"`
		UploadedAt string   `json:"uploaded_at"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := UserIDFromCtx(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		orders, err := svc.ListByUser(r.Context(), uid)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(orders) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		sort.Slice(orders, func(i, j int) bool { return orders[i].UploadedAt.After(orders[j].UploadedAt) })

		resp := make([]orderDTO, 0, len(orders))
		for _, o := range orders {
			var accrual *float64
			if o.Accrual != nil {
				v := o.Accrual.InexactFloat64()
				accrual = &v
			}
			resp = append(resp, orderDTO{
				Number:     o.Number,
				Status:     o.Status,
				Accrual:    accrual,
				UploadedAt: o.UploadedAt.Format(time.RFC3339),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
