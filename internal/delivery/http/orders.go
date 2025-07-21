package http

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Hobrus/gophermarket/internal/domain"
)

type orderDTO struct {
	Number     string   `json:"number"`
	Status     string   `json:"status"`
	Accrual    *float64 `json:"accrual,omitempty"`
	UploadedAt string   `json:"uploaded_at"`
}

// ListService defines method required for listing user orders.
type ListService interface {
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Order, error)
}

// NewOrdersRouter creates chi router with user orders endpoints.
func NewOrdersRouter(svc ListService) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/user/orders", ListOrders(svc))
	return r
}

// listOrders returns list of user's orders
// @Summary List user orders
// @Success 200 {array} orderDTO
// @Success 204 {string} string "No Content"
// @Success 401 {string} string "Unauthorized"
// @Success 500 {string} string "Internal Server Error"
// @Router /api/user/orders [get]
func ListOrders(svc ListService) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := UserIDFromCtx(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		q := r.URL.Query()
		limit := 50
		if v := q.Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				if n > 100 {
					n = 100
				}
				limit = n
			}
		}
		offset := 0
		if v := q.Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}

		orders, err := svc.ListByUser(r.Context(), uid, limit, offset)
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
