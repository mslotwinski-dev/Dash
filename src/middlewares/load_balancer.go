package middlewares

import (
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mslotwinski-dev/dash/src/backend"
	"github.com/mslotwinski-dev/dash/src/utils"
)

type LoadBalancer struct {
	backends []*backend.Backend
	counter  uint64
	strategy string
	mu       sync.RWMutex
}

func NewLoadBalancer(backends []*backend.Backend, strategy string) *LoadBalancer {
	return &LoadBalancer{
		backends: backends,
		counter:  0,
		strategy: strategy,
	}
}

func (lb *LoadBalancer) GetNextBackend(r *http.Request) *backend.Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// 1. Sticky Sessions: sprawdzamy ciasteczko
	if cookie, err := r.Cookie("dash_sticky"); err == nil && cookie.Value != "" {
		for _, b := range lb.backends {
			if b.IsAlive() && b.URL.String() == cookie.Value {
				return b
			}
		}
	}

	var target *backend.Backend

	switch lb.strategy {
	case "least-conn":
		target = lb.getLeastConn()
	case "weighted":
		target = lb.getWeighted()
	default:
		target = lb.getRoundRobin()
	}

	return target
}

func (lb *LoadBalancer) getRoundRobin() *backend.Backend {
	n := uint64(len(lb.backends))
	for i := uint64(0); i < n; i++ {
		idx := atomic.AddUint64(&lb.counter, 1) - 1
		targetBackend := lb.backends[idx%n]
		if targetBackend.IsAlive() {
			return targetBackend
		}
	}
	return nil
}

func (lb *LoadBalancer) getLeastConn() *backend.Backend {
	var best *backend.Backend
	minConn := int64(-1)

	for _, b := range lb.backends {
		if !b.IsAlive() {
			continue
		}
		conns := atomic.LoadInt64(&b.ActiveConnections)
		if minConn == -1 || conns < minConn {
			minConn = conns
			best = b
		}
	}
	return best
}

func (lb *LoadBalancer) getWeighted() *backend.Backend {
	totalWeight := 0
	for _, b := range lb.backends {
		if b.IsAlive() {
			totalWeight += b.Weight
		}
	}

	if totalWeight == 0 {
		return nil
	}

	r := rand.Intn(totalWeight)
	for _, b := range lb.backends {
		if !b.IsAlive() {
			continue
		}
		r -= b.Weight
		if r < 0 {
			return b
		}
	}
	return nil
}

func (lb *LoadBalancer) StartHealthCheck() {
	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for range ticker.C {
			lb.mu.RLock()
			currentBackends := make([]*backend.Backend, len(lb.backends))
			copy(currentBackends, lb.backends)
			lb.mu.RUnlock()

			for _, b := range currentBackends {
				client := http.Client{Timeout: 2 * time.Second}

				resp, err := client.Get(b.URL.String())

				if err != nil || resp.StatusCode != http.StatusOK {
					if b.IsAlive() {
						b.SetAlive(false)
						utils.Error("Backend %s PRZESTAŁ DZIAŁAĆ!", b.URL.String())
					}
				} else {
					if !b.IsAlive() {
						b.SetAlive(true)
						utils.Info("Backend %s URUCHOMIONY PONOWNIE!", b.URL.String())
					}
					resp.Body.Close()
				}
			}
		}
	}()
}

func (lb *LoadBalancer) GetBackends() []*backend.Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.backends
}

func (lb *LoadBalancer) AddBackend(b *backend.Backend) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.backends = append(lb.backends, b)
}

func (lb *LoadBalancer) RemoveBackend(target string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	for i, b := range lb.backends {
		if b.URL.String() == target {
			lb.backends = append(lb.backends[:i], lb.backends[i+1:]...)
			break
		}
	}
}
