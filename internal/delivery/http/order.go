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
