package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/shopspring/decimal"
)

// BalanceService defines methods required to obtain user balance.
type BalanceService interface {
	GetBalance(ctx context.Context, userID int64) (decimal.Decimal, decimal.Decimal, error)
}

func balance(balanceSvc BalanceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromCtx(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		current, withdrawn, err := balanceSvc.GetBalance(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp := struct {
			Current   float64 `json:"current"`
			Withdrawn float64 `json:"withdrawn"`
		}{
			Current:   current.InexactFloat64(),
			Withdrawn: withdrawn.InexactFloat64(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
