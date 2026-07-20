package middlewares

import (
	"compress/gzip"
	"net/http"
	"strings"
)

type compressResponseWriter struct {
	http.ResponseWriter
	gzWriter     *gzip.Writer
	bypass       bool
	wroteHeaders bool
}

func (w *compressResponseWriter) WriteHeader(code int) {
	if w.wroteHeaders {
		return
	}
	w.wroteHeaders = true

	if w.Header().Get("Content-Encoding") != "" {
		w.bypass = true
	} else {
		contentType := w.Header().Get("Content-Type")
		// Kompresujemy tylko tekstowe i JSON-owe odpowiedzi
		if contentType == "" || strings.Contains(contentType, "text/") || strings.Contains(contentType, "application/json") || strings.Contains(contentType, "application/javascript") || strings.Contains(contentType, "image/svg+xml") {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Del("Content-Length")
			w.Header().Add("Vary", "Accept-Encoding")
		} else {
			w.bypass = true
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *compressResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeaders {
		w.WriteHeader(http.StatusOK)
	}
	if w.bypass {
		return w.ResponseWriter.Write(b)
	}
	return w.gzWriter.Write(b)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gzWriter := gzip.NewWriter(w)
		defer gzWriter.Close()

		gzw := &compressResponseWriter{
			ResponseWriter: w,
			gzWriter:       gzWriter,
		}

		next.ServeHTTP(gzw, r)
	})
}
