package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Hobrus/gophermarket/internal/domain"
	"github.com/Hobrus/gophermarket/pkg/luhn"
	"github.com/shopspring/decimal"
)

// WithdrawalService defines method required for balance withdrawal.
type WithdrawalService interface {
	Withdraw(ctx context.Context, userID int64, number string, amount decimal.Decimal) error
}

type reqDTO struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

// Withdraw returns handler for POST /api/user/balance/withdraw.
// @Summary Withdraw user balance
// @Param request body reqDTO true "Withdraw info"
// @Success 200 {string} string "OK"
// @Success 400 {string} string "Bad Request"
// @Success 401 {string} string "Unauthorized"
// @Success 402 {string} string "Payment Required"
// @Success 422 {string} string "Unprocessable Entity"
// @Success 500 {string} string "Internal Server Error"
// @Router /api/user/balance/withdraw [post]
func Withdraw(svc WithdrawalService) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		uid, ok := UserIDFromCtx(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var req reqDTO
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !luhn.IsValid(req.Order) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		err := svc.Withdraw(r.Context(), uid, req.Order, decimal.NewFromFloat(req.Sum))
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrInsufficientFunds):
				w.WriteHeader(http.StatusPaymentRequired)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
