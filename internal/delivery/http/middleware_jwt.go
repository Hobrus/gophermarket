package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

// ctxKey is context key type for storing values.
type ctxKey string

const userIDKey ctxKey = "user_id"

// JWT parses AuthToken cookie and validates JWT.
// On success user id is stored in request context.
func JWT(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie("AuthToken")
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(c.Value, func(t *jwt.Token) (interface{}, error) {
				if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, errors.New("unexpected signing method")
				}
				return secret, nil
			})
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok || !token.Valid {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			sub, ok := claims["sub"].(float64)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, int64(sub))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromCtx extracts user id from context.
func UserIDFromCtx(ctx context.Context) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}
