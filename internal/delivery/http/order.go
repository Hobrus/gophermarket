package http

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Hobrus/gophermarket/pkg/luhn"
)

// UploadService defines methods required for order upload operations.
type UploadService interface {
	Add(ctx context.Context, userID int64, number string) (errConflictSelf, errConflictOther, err error)
}

// NewOrderRouter creates router with order upload endpoint.
func NewOrderRouter(svc UploadService) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/user/orders", uploadOrder(svc))
	return r
}

// uploadOrder uploads new order number
// @Summary Upload order number
// @Param number body string true "Order number"
// @Success 202 {string} string "Accepted"
// @Success 200 {string} string "Already uploaded"
// @Success 400 {string} string "Bad Request"
// @Success 401 {string} string "Unauthorized"
// @Success 409 {string} string "Conflict"
// @Success 422 {string} string "Unprocessable Entity"
// @Success 500 {string} string "Internal Server Error"
// @Router /api/user/orders [post]
func uploadOrder(svc UploadService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromCtx(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		number := strings.TrimSpace(string(body))
		if !luhn.IsValid(number) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		errSelf, errOther, err := svc.Add(r.Context(), userID, number)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if errSelf != nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		if errOther != nil {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}
