package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGzip_Decompress(t *testing.T) {
	called := false
	h := Gzip(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if string(b) != "hello" {
			t.Fatalf("unexpected body %q", b)
		}
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write([]byte("hello"))
	gz.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if !called {
		t.Fatal("handler not called")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
}

func TestGzip_DecompressBad(t *testing.T) {
	h := Gzip(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler called")
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGzip_Compress(t *testing.T) {
	h := Gzip(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("missing encoding header")
	}
	zr, err := gzip.NewReader(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatalf("new reader: %v", err)
	}
	data, _ := io.ReadAll(zr)
	if string(data) != "world" {
		t.Fatalf("unexpected body %q", data)
	}
}

func TestGzip_NoCompress(t *testing.T) {
	h := Gzip(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "" {
		t.Fatalf("unexpected encoding header %s", w.Header().Get("Content-Encoding"))
	}
	if w.Body.String() != "plain" {
		t.Fatalf("unexpected body %q", w.Body.String())
	}
}
