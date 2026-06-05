package services

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/mslotwinski-dev/dash/src/middlewares"
	"github.com/mslotwinski-dev/dash/src/utils"
)

type responseWriterDelegator struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterDelegator) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

var (
	// PROMETHEUS ELEGANCKA POPRAWKA: Prosty licznik globalny, idealny do odczytu w WebSockecie
	dashTotalRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "dash_global_requests_total",
			Help: "Suma wszystkich żądań obsłużonych od startu serwera",
		},
	)

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
	prometheus.MustRegister(dashTotalRequests)
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

func MetricsMiddleware(hub *WsHub, lb *middlewares.LoadBalancer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriterDelegator{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start).Seconds()

			endpoint := r.URL.Path
			if len(endpoint) >= 4 && endpoint[:4] == "/api" {
				endpoint = "/api/*"
			}

			// Inkrementujemy zarówno metryki szczegółowe, jak i nasz nowy licznik globalny
			dashTotalRequests.Inc()
			httpRequestsTotal.WithLabelValues(r.Method, endpoint, strconv.Itoa(rw.statusCode)).Inc()
			httpRequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)

			utils.Info("[REQUEST] %s %s | Status: %d | Czas: %v", r.Method, r.URL.Path, rw.statusCode, time.Since(start))

			var active, dead []string
			for _, b := range lb.GetBackends() {
				if b.IsAlive() {
					active = append(active, b.URL.String())
				} else {
					dead = append(dead, b.URL.String())
				}
			}

			// POBIERANIE DANYCH Z PROMETHEUSA:
			// Wyciągamy aktualny stan licznika Prometheusa do czystej zmiennej uint64
			var m dto.Metric
			_ = dashTotalRequests.Write(&m)
			currentTotal := uint64(m.GetCounter().GetValue())

			go func() {
				hub.Broadcast <- DashboardStats{
					TotalRequests:  currentTotal, // Wykorzystane bezpośrednio z Prometheusa!
					ActiveBackends: active,
					DeadBackends:   dead,
					LastRequest:    r.Method + " " + r.URL.Path + " [" + strconv.Itoa(rw.statusCode) + "]",
				}
			}()
		})
	}
}
