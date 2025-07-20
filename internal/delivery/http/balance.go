package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Hobrus/gophermarket/internal/domain"
)

type respDTO struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// BalanceService defines method required to get user balance.
type BalanceService interface {
	GetBalance(ctx context.Context, userID int64) (domain.Balance, error)
}

// Balance returns handler for GET /api/user/balance.
// @Summary Get user balance
// @Success 200 {object} respDTO
// @Success 401 {string} string "Unauthorized"
// @Success 500 {string} string "Internal Server Error"
// @Router /api/user/balance [get]
func Balance(svc BalanceService) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := UserIDFromCtx(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		bal, err := svc.GetBalance(r.Context(), uid)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp := respDTO{
			Current:   bal.Current.InexactFloat64(),
			Withdrawn: bal.Withdrawn.InexactFloat64(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
