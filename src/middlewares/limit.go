package middlewares

import (
	"net"
	"net/http"
	"sync"

	"github.com/mslotwinski-dev/dash/src/utils"
	"golang.org/x/time/rate"
)

// IPlimiter przechowuje limitery dla poszczególnych adresów IP
type IPlimiter struct {
	sync.RWMutex
	limiters map[string]*rate.Limiter
	r        rate.Limit // jak szybko odnawiają się tokeny (na sekundę)
	b        int        // maksymalna pojemność kubełka (burst)
}

func NewIPLimiter(r rate.Limit, b int) *IPlimiter {
	return &IPlimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (i *IPlimiter) GetLimiter(ip string) *rate.Limiter {
	i.Lock()
	defer i.Unlock()

	limiter, exists := i.limiters[ip]
	if !exists {
		// Tworzymy nowy limiter: i.r to prędkość, i.b to pojemność kubełka
		limiter = rate.NewLimiter(i.r, i.b)
		i.limiters[ip] = limiter
	}

	return limiter
}

// RateLimitMiddleware sprawdza, czy dany użytkownik nie przekroczył limitu zapytań
func RateLimitMiddleware(limiter *IPlimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wyciągamy adres IP użytkownika (odcinamy port po dwukropku)
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Pobieramy lub tworzymy limiter dla tego konkretnego IP
		userLimiter := limiter.GetLimiter(ip)

		// Sprawdzamy, czy użytkownik ma jeszcze dostępny token
		if !userLimiter.Allow() {
			utils.Warn("[RATE LIMIT] Przekroczono limit zapytań dla IP: %s! Zwracam 429", ip)

			// Zwracamy standardowy kod HTTP 429 Too Many Requests
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "Too many requests. Please slow down."}`))
			return
		}

		// Jeśli wszystko ok, puszczamy żądanie dalej
		next.ServeHTTP(w, r)
	})
}
