package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipReadCloser struct {
	io.Reader
	gz *gzip.Reader
	rc io.Closer
}

func (g *gzipReadCloser) Close() error {
	_ = g.gz.Close()
	if g.rc != nil {
		return g.rc.Close()
	}
	return nil
}

// Gzip returns middleware that transparently decompresses request bodies
// with Content-Encoding: gzip and compresses responses if the client
// sends Accept-Encoding containing "gzip".
func Gzip(level int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Encoding") == "gzip" {
				gz, err := gzip.NewReader(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				r.Body = &gzipReadCloser{Reader: gz, gz: gz, rc: r.Body}
				r.Header.Del("Content-Encoding")
			}

			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				gz, err := gzip.NewWriterLevel(w, level)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				defer gz.Close()

				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Del("Content-Length")

				gw := &gzipResponseWriter{ResponseWriter: w, Writer: gz}
				next.ServeHTTP(gw, r)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	io.Writer
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	return w.Writer.Write(p)
}
