package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/Hobrus/gophermarket/internal/domain"
	"github.com/Hobrus/gophermarket/pkg/luhn"
)

// BalanceService defines methods required for balance operations.
type BalanceService interface {
	Withdraw(ctx context.Context, userID int64, order string, amount decimal.Decimal) error
}

func NewBalanceRouter(s BalanceService) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/user/balance/withdraw", withdraw(s))
	return r
}

type withdrawRequest struct {
	Order string          `json:"order"`
	Sum   decimal.Decimal `json:"sum"`
}

func withdraw(s BalanceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req withdrawRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !luhn.IsValid(req.Order) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		id, ok := UserIDFromCtx(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := s.Withdraw(r.Context(), id, req.Order, req.Sum); err != nil {
			if errors.Is(err, domain.ErrInsufficientBalance) {
				w.WriteHeader(http.StatusPaymentRequired)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
