package http

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Hobrus/gophermarket/internal/domain"
)

type stubAuth struct {
	registerFunc func(ctx context.Context, login, password string) (string, error)
	loginFunc    func(ctx context.Context, login, password string) (string, error)
}

func (s *stubAuth) Register(ctx context.Context, login, password string) (string, error) {
	return s.registerFunc(ctx, login, password)
}

func (s *stubAuth) Login(ctx context.Context, login, password string) (string, error) {
	return s.loginFunc(ctx, login, password)
}

func TestRegister_Success(t *testing.T) {
	auth := &stubAuth{registerFunc: func(ctx context.Context, login, password string) (string, error) {
		if login != "user" || password != "pass" {
			t.Fatalf("unexpected args %s %s", login, password)
		}
		return "token", nil
	}}
	router := NewRouter(auth)

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(`{"login":"user","password":"pass"}`))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var c *http.Cookie
	for _, ck := range res.Cookies() {
		if ck.Name == "AuthToken" {
			c = ck
			break
		}
	}
	if c == nil {
		t.Fatal("cookie missing")
	}
	if c.Value != "token" || !c.HttpOnly || c.Secure || c.Path != "/" {
		t.Fatal("cookie properties incorrect")
	}
}

func TestRegister_Conflict(t *testing.T) {
	auth := &stubAuth{registerFunc: func(ctx context.Context, login, password string) (string, error) {
		return "", domain.ErrConflictSelf
	}}
	router := NewRouter(auth)

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(`{"login":"a","password":"b"}`))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
}

func TestRegister_BadRequest(t *testing.T) {
	auth := &stubAuth{registerFunc: func(ctx context.Context, login, password string) (string, error) {
		return "", errors.New("should not be called")
	}}
	router := NewRouter(auth)

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString("{"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestLogin_Unauthorized(t *testing.T) {
	auth := &stubAuth{loginFunc: func(ctx context.Context, login, password string) (string, error) {
		return "", errors.New("invalid credentials")
	}}
	router := NewRouter(auth)

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(`{"login":"u","password":"p"}`))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

func TestLogin_Success(t *testing.T) {
	auth := &stubAuth{loginFunc: func(ctx context.Context, login, password string) (string, error) {
		if login != "user" || password != "pass" {
			t.Fatalf("unexpected args %s %s", login, password)
		}
		return "tok", nil
	}}
	router := NewRouter(auth)

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(`{"login":"user","password":"pass"}`))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var c *http.Cookie
	for _, ck := range res.Cookies() {
		if ck.Name == "AuthToken" {
			c = ck
			break
		}
	}
	if c == nil {
		t.Fatal("cookie missing")
	}
	if c.Value != "tok" || !c.HttpOnly || c.Secure || c.Path != "/" {
		t.Fatal("cookie properties incorrect")
	}
}
