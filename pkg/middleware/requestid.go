package middleware

import (
	"net/http"

	"github.com/Hobrus/gophermarket/pkg/logger"
	"github.com/google/uuid"
)

// RequestID generates a new UUID v4 for each request and stores it in the
// request context and the X-Request-ID header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		w.Header().Set("X-Request-ID", id)
		ctx := logger.WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
