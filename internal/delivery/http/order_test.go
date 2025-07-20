package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Hobrus/gophermarket/internal/domain"
)

type stubOrderService struct {
	addFunc func(ctx context.Context, userID int64, number string) (error, error, error)
}

func (s *stubOrderService) Add(ctx context.Context, userID int64, number string) (error, error, error) {
	return s.addFunc(ctx, userID, number)
}

func TestUploadOrder(t *testing.T) {
	validNum := "79927398713"
	tests := []struct {
		name       string
		user       bool
		body       string
		gzip       bool
		noCompress bool
		addFn      func(ctx context.Context, userID int64, number string) (error, error, error)
		status     int
	}{
		{
			name:   "unauthorized",
			user:   false,
			status: http.StatusUnauthorized,
		},
		{
			name: "invalid number",
			user: true,
			body: "123",
			addFn: func(ctx context.Context, userID int64, number string) (error, error, error) {
				t.Errorf("should not call add")
				return nil, nil, nil
			},
			status: http.StatusUnprocessableEntity,
		},
		{
			name: "conflict self",
			user: true,
			body: validNum,
			addFn: func(ctx context.Context, userID int64, number string) (error, error, error) {
				if number != validNum || userID != 1 {
					t.Fatalf("unexpected args")
				}
				return domain.ErrConflictSelf, nil, nil
			},
			status: http.StatusOK,
		},
		{
			name: "conflict other",
			user: true,
			body: validNum,
			addFn: func(ctx context.Context, userID int64, number string) (error, error, error) {
				return nil, domain.ErrConflictOther, nil
			},
			status: http.StatusConflict,
		},
		{
			name: "service error",
			user: true,
			body: validNum,
			addFn: func(ctx context.Context, userID int64, number string) (error, error, error) {
				return nil, nil, errors.New("fail")
			},
			status: http.StatusInternalServerError,
		},
		{
			name: "success",
			user: true,
			body: validNum,
			addFn: func(ctx context.Context, userID int64, number string) (error, error, error) {
				if number != validNum || userID != 1 {
					t.Fatalf("unexpected args")
				}
				return nil, nil, nil
			},
			status: http.StatusAccepted,
		},
		{
			name: "gzip",
			user: true,
			body: validNum,
			gzip: true,
			addFn: func(ctx context.Context, userID int64, number string) (error, error, error) {
				if number != validNum {
					t.Fatalf("unexpected num %s", number)
				}
				return nil, nil, nil
			},
			status: http.StatusAccepted,
		},
		{
			name:       "bad gzip",
			user:       true,
			gzip:       true,
			noCompress: true,
			body:       "notgzip",
			addFn: func(ctx context.Context, userID int64, number string) (error, error, error) {
				t.Errorf("should not call add")
				return nil, nil, nil
			},
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		svc := &stubOrderService{addFunc: tt.addFn}
		router := NewOrdersRouter(svc)

		var buf bytes.Buffer
		if tt.gzip && !tt.noCompress {
			gz := gzip.NewWriter(&buf)
			io.WriteString(gz, tt.body)
			gz.Close()
		} else {
			buf.WriteString(tt.body)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/user/orders", &buf)
		if tt.gzip {
			req.Header.Set("Content-Encoding", "gzip")
		}
		if tt.user {
			req = req.WithContext(context.WithValue(req.Context(), userIDKey, int64(1)))
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Result().StatusCode != tt.status {
			t.Errorf("%s: expected %d, got %d", tt.name, tt.status, w.Result().StatusCode)
		}
	}
}
