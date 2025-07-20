package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWT_NoToken(t *testing.T) {
	handlerCalled := false
	mw := JWT([]byte("secret"))
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if handlerCalled {
		t.Fatal("handler should not be called")
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

func TestJWT_WithToken(t *testing.T) {
	var gotID int64
	mw := JWT([]byte("secret"))
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := UserIDFromCtx(r.Context())
		if !ok {
			t.Fatal("user id missing")
		}
		gotID = id
		w.WriteHeader(http.StatusOK)
	}))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": int64(42),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "AuthToken", Value: tokenStr})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if gotID != 42 {
		t.Fatalf("expected id 42, got %d", gotID)
	}
}
