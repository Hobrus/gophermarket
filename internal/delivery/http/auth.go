package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Hobrus/gophermarket/internal/domain"
)

// AuthService defines methods required for user authentication.
type AuthService interface {
	Register(ctx context.Context, login, password string) (string, error)
	Login(ctx context.Context, login, password string) (string, error)
}

type credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// NewRouter creates chi router with authentication endpoints.
func NewRouter(auth AuthService) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/user/register", register(auth))
	r.Post("/api/user/login", login(auth))
	return r
}

func register(auth AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		token, err := auth.Register(r.Context(), creds.Login, creds.Password)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrConflictSelf):
				w.WriteHeader(http.StatusConflict)
			case err.Error() == "login too short":
				w.WriteHeader(http.StatusBadRequest)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "AuthToken",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
		})
		w.WriteHeader(http.StatusOK)
	}
}

func login(auth AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		token, err := auth.Login(r.Context(), creds.Login, creds.Password)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrNotFound), err.Error() == "invalid credentials":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "AuthToken",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
		})
		w.WriteHeader(http.StatusOK)
	}
}
