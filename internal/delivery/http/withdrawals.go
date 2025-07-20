package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Hobrus/gophermarket/internal/domain"
)

// WithdrawalRepo defines methods required to fetch withdrawals.
type WithdrawalRepo interface {
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Withdrawal, error)
}

type respItem struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

// Withdrawals returns handler for GET /api/user/withdrawals.
// @Summary List user withdrawals
// @Success 200 {array} respItem
// @Success 204 {string} string "No Content"
// @Success 401 {string} string "Unauthorized"
// @Success 500 {string} string "Internal Server Error"
// @Router /api/user/withdrawals [get]
func Withdrawals(repo WithdrawalRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromCtx(r.Context())
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

		list, err := repo.ListByUser(r.Context(), userID, limit, offset)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(list) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		resp := make([]respItem, len(list))
		for i, it := range list {
			resp[i] = respItem{
				Order:       it.Number,
				Sum:         it.Amount.InexactFloat64(),
				ProcessedAt: it.ProcessedAt.Format(time.RFC3339),
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
