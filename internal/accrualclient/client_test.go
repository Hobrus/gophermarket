package accrualclient

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestHTTPClient_Get(t *testing.T) {
	type want struct {
		status  string
		accrual *decimal.Decimal
		retry   time.Duration
		err     bool
	}

	dec := decimal.NewFromInt(10)

	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    want
	}{
		{
			name: "ok gzip",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Accept-Encoding") != "gzip" {
					t.Errorf("missing Accept-Encoding header")
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Content-Encoding", "gzip")
				w.WriteHeader(http.StatusOK)
				gz := gzip.NewWriter(w)
				json.NewEncoder(gz).Encode(map[string]any{
					"order":   "42",
					"status":  "PROCESSED",
					"accrual": dec,
				})
				gz.Close()
			},
			want: want{status: "PROCESSED", accrual: &dec},
		},
		{
			name: "no content",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Accept-Encoding") != "gzip" {
					t.Errorf("missing Accept-Encoding header")
				}
				w.WriteHeader(http.StatusNoContent)
			},
			want: want{},
		},
		{
			name: "retry",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Accept-Encoding") != "gzip" {
					t.Errorf("missing Accept-Encoding header")
				}
				w.Header().Set("Retry-After", "2")
				w.WriteHeader(http.StatusTooManyRequests)
			},
			want: want{retry: 2 * time.Second},
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			want: want{err: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			c := New(srv.URL)
			status, accrual, retry, err := c.Get(context.Background(), "42")

			if tt.want.err {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if status != tt.want.status {
				t.Errorf("status %s != %s", status, tt.want.status)
			}
			if (accrual == nil) != (tt.want.accrual == nil) {
				t.Fatalf("accrual nil mismatch")
			}
			if accrual != nil && !accrual.Equal(*tt.want.accrual) {
				t.Errorf("accrual %s != %s", accrual, tt.want.accrual)
			}
			if retry != tt.want.retry {
				t.Errorf("retry %v != %v", retry, tt.want.retry)
			}
		})
	}
}

func TestHTTPClient_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL)

	start := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Get(context.Background(), "42")
		}()
	}
	wg.Wait()

	if time.Since(start) < 2*time.Second {
		t.Fatalf("expected duration >= 2s, got %v", time.Since(start))
	}
}
