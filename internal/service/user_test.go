package service

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Hobrus/gophermarket/internal/domain"
	"github.com/Hobrus/gophermarket/pkg/crypto"
)

type stubRepo struct {
	createFunc     func(ctx context.Context, login, hash string) (int64, error)
	getByLoginFunc func(ctx context.Context, login string) (domain.User, error)
}

func (s *stubRepo) Create(ctx context.Context, login, hash string) (int64, error) {
	return s.createFunc(ctx, login, hash)
}
func (s *stubRepo) GetByLogin(ctx context.Context, login string) (domain.User, error) {
	return s.getByLoginFunc(ctx, login)
}

func parseToken(t *testing.T, tokenStr string, secret []byte) jwt.MapClaims {
	t.Helper()
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		t.Fatalf("invalid token")
	}
	return claims
}

func TestAuthService_RegisterSuccess(t *testing.T) {
	repo := &stubRepo{createFunc: func(ctx context.Context, login, hash string) (int64, error) {
		if login != "user" || hash == "" {
			t.Fatalf("unexpected create args %s %s", login, hash)
		}
		return 1, nil
	}}
	svc := NewAuthService(repo, []byte("secret"))

	tokenStr, err := svc.Register(context.Background(), "user", "pass")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	claims := parseToken(t, tokenStr, []byte("secret"))
	if sub, ok := claims["sub"].(float64); !ok || int64(sub) != 1 {
		t.Errorf("unexpected sub %v", claims["sub"])
	}
	if claims["login"] != "user" {
		t.Errorf("unexpected login claim %v", claims["login"])
	}
}

func TestAuthService_RegisterConflict(t *testing.T) {
	repo := &stubRepo{createFunc: func(ctx context.Context, login, hash string) (int64, error) {
		return 0, domain.ErrConflictSelf
	}}
	svc := NewAuthService(repo, []byte("secret"))

	if _, err := svc.Register(context.Background(), "user", "pass"); !errors.Is(err, domain.ErrConflictSelf) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestAuthService_LoginWrongPassword(t *testing.T) {
	hash, _ := crypto.HashPassword("pass")
	repo := &stubRepo{getByLoginFunc: func(ctx context.Context, login string) (domain.User, error) {
		return domain.User{ID: 1, Login: login, PasswordHash: hash}, nil
	}}
	svc := NewAuthService(repo, []byte("secret"))

	if _, err := svc.Login(context.Background(), "user", "wrong"); err == nil {
		t.Fatal("expected error for wrong password")
	}
}
