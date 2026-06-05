package src

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/mslotwinski-dev/dash/src/utils"
)

// Tworzymy strukturę niestandardowego ResponseWritera, żeby móc przechwycić Status Code (np. 200, 404)
type responseWriterDelegator struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterDelegator) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Rejestrujemy metryki w Prometheusie
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dash_http_requests_total",
			Help: "Całkowita liczba żądań HTTP obsłużonych przez Dash",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dash_http_request_duration_seconds",
			Help:    "Czas odpowiedzi serwera Dash w sekundach",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	// Rejestracja metryk podczas startu aplikacji
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// MetricsMiddleware przechwytuje dane o każdym żądaniu dla Prometheusa
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Domyślnie zakładamy status 200, jeśli handler sam go nie ustawi
		rw := &responseWriterDelegator{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()

		// Określamy ogólny endpoint dla metryk (np. grupujemy całe api)
		endpoint := r.URL.Path
		if len(endpoint) >= 4 && endpoint[:4] == "/api" {
			endpoint = "/api/*" // nie chcemy tysięcy osobnych metryk dla każdego ID usera
		}

		// Zapisujemy dane do Prometheusa
		httpRequestsTotal.WithLabelValues(r.Method, endpoint, strconv.Itoa(rw.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)

		// Logujemy też tradycyjnie do konsoli za pomocą Twojego utils
		utils.Info("[REQUEST] %s %s | Status: %d | Czas: %v", r.Method, r.URL.Path, rw.statusCode, time.Since(start))
	})
}
