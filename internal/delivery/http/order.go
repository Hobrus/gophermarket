package http

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Hobrus/gophermarket/pkg/luhn"
)

// OrderService defines methods required for order operations.
type OrderService interface {
	Add(ctx context.Context, userID int64, number string) (errConflictSelf, errConflictOther, err error)
}

// NewOrdersRouter creates router with order endpoints.
func NewOrdersRouter(svc OrderService) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/user/orders", uploadOrder(svc))
	return r
}

func uploadOrder(svc OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromCtx(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var reader io.Reader = r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			defer gz.Close()
			reader = gz
		}
		body, err := io.ReadAll(reader)
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
