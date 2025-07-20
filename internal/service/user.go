package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Hobrus/gophermarket/internal/repository"
	"github.com/Hobrus/gophermarket/pkg/crypto"
)

// AuthService provides user registration and authentication logic.
type AuthService struct {
	repo      repository.UserRepo
	jwtSecret []byte
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(repo repository.UserRepo, secret []byte) *AuthService {
	return &AuthService{repo: repo, jwtSecret: secret}
}

// Register registers a new user and returns JWT token.
func (s *AuthService) Register(ctx context.Context, login, password string) (string, error) {
	if len(login) < 3 {
		return "", errors.New("login too short")
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return "", err
	}
	id, err := s.repo.Create(ctx, login, hash)
	if err != nil {
		return "", err
	}
	return s.issueToken(id, login)
}

// Login authenticates user and returns JWT token.
func (s *AuthService) Login(ctx context.Context, login, password string) (string, error) {
	u, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return "", err
	}
	if err := crypto.ComparePassword(u.PasswordHash, password); err != nil {
		return "", errors.New("invalid credentials")
	}
	return s.issueToken(u.ID, u.Login)
}

func (s *AuthService) issueToken(userID int64, login string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"login": login,
		"exp":   time.Now().Add(72 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
