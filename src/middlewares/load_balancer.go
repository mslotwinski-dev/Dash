package middlewares

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/mslotwinski-dev/dash/src/backend"
	"github.com/mslotwinski-dev/dash/src/utils"
)

type LoadBalancer struct {
	backends []*backend.Backend
	counter  uint64
}

func NewLoadBalancer(backends []*backend.Backend) *LoadBalancer {
	return &LoadBalancer{
		backends: backends,
		counter:  0,
	}
}

func (lb *LoadBalancer) GetNextBackend() *backend.Backend {
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

func (lb *LoadBalancer) StartHealthCheck() {
	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for range ticker.C {
			for _, b := range lb.backends {
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
	return lb.backends
}
