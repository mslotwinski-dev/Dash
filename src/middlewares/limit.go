package middlewares

import (
	"net"
	"net/http"
	"sync"

	"github.com/mslotwinski-dev/dash/src/utils"
	"golang.org/x/time/rate"
)

type IPLimiter struct {
	sync.RWMutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func NewIPLimiter(r rate.Limit, b int) *IPLimiter {
	return &IPLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (i *IPLimiter) GetLimiter(ip string) *rate.Limiter {
	i.Lock()
	defer i.Unlock()

	limiter, exists := i.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.limiters[ip] = limiter
	}

	return limiter
}

func RateLimitMiddleware(limiter *IPLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		userLimiter := limiter.GetLimiter(ip)

		if !userLimiter.Allow() {
			utils.Warn("[RATE LIMIT] Przekroczono limit zapytań dla IP: %s! Zwracam 429", ip)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "Too many requests. Please slow down."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
