package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Hobrus/gophermarket/internal/domain"
)

// WithdrawalRepo defines methods required to fetch withdrawals.
type WithdrawalRepo interface {
	ListByUser(ctx context.Context, userID int64) ([]domain.Withdrawal, error)
}

// Withdrawals returns handler for GET /api/user/withdrawals.
func Withdrawals(repo WithdrawalRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromCtx(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		list, err := repo.ListByUser(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(list) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		type respItem struct {
			Order       string  `json:"order"`
			Sum         float64 `json:"sum"`
			ProcessedAt string  `json:"processed_at"`
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
