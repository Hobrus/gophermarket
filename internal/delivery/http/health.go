package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// DBPinger describes minimal database ping capability.
type DBPinger interface {
	Ping(ctx context.Context) error
}

// NewHealthRouter creates chi router with health check endpoints.
func NewHealthRouter(db DBPinger) http.Handler {
	r := chi.NewRouter()
	r.Get("/live", live())
	r.Get("/ready", ready(db))
	return r
}

// live returns 200 OK.
// @Summary Liveness check
// @Success 200 {string} string "OK"
// @Router /health/live [get]
func live() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

// ready checks database connectivity.
// @Summary Readiness check
// @Success 200 {string} string "OK"
// @Failure 500 {string} string "Internal Server Error"
// @Router /health/ready [get]
func ready(db DBPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
