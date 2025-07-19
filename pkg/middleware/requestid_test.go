package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID_Header(t *testing.T) {
	called := false
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if id := w.Header().Get("X-Request-ID"); id == "" {
			t.Error("request id header missing in handler")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if !called {
		t.Fatal("handler not called")
	}
	if id := w.Header().Get("X-Request-ID"); id == "" {
		t.Error("request id header missing")
	}
}
