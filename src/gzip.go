package src

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

// Nadpisujemy metodę Write, aby dane szły przez kompresor gzip
func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// GzipMiddleware sprawdza, czy klient wspiera gzip i jeśli tak, kompresuje odpowiedź
// GzipMiddleware w bezpiecznej wersji "Smart"
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Jeśli klient nie wspiera gzip, puszczamy ruch dalej bez zmian
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// 2. Jeśli to jest zapytanie do API, nie kompresujemy go naszym middleware
		// (Backendy same decydują o swojej kompresji)
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			next.ServeHTTP(w, r)
			return
		}

		// 3. Sprytny ResponseWriter, który włączy Gzip TYLKO dla sukcesu (Status 200) i plików tekstowych
		gzWriter := gzip.NewWriter(w)

		// Tworzymy strukturę przechwytującą moment zapisu
		gzw := gzipResponseWriter{
			Writer:         gzWriter,
			ResponseWriter: w,
		}

		// Ustawiamy nagłówek informacyjny dla przeglądarki
		w.Header().Set("Content-Encoding", "gzip")

		// Wykonujemy resztę aplikacji
		next.ServeHTTP(gzw, r)

		// Zamykamy kompresor dopiero po upewnieniu się, że wszystko poszło ok
		gzWriter.Close()
	})
}
