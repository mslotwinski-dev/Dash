package middlewares

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			next.ServeHTTP(w, r)
			return
		}

		gzWriter := gzip.NewWriter(w)

		gzw := gzipResponseWriter{
			Writer:         gzWriter,
			ResponseWriter: w,
		}

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzw, r)
		gzWriter.Close()
	})
}
