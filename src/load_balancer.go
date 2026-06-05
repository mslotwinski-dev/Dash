package src

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/mslotwinski-dev/dash/src/utils"
)

type LoadBalancer struct {
	backends []*Backend
	counter  uint64
}

// GetNextBackend wybiera kolejny DOSTĘPNY (żywy) serwer metodą Round-Robin
func (lb *LoadBalancer) GetNextBackend() *Backend {
	n := uint64(len(lb.backends))

	// Próbujemy znaleźć żywy serwer, wykonując maksymalnie 'n' prób
	for i := uint64(0); i < n; i++ {
		idx := atomic.AddUint64(&lb.counter, 1) - 1
		targetBackend := lb.backends[idx%n]

		if targetBackend.IsAlive() {
			return targetBackend
		}
	}
	return nil // Wszystkie backendy leżą!
}

// HealthCheck uruchamia pętlę w tle sprawdzającą stan serwerów
func (lb *LoadBalancer) StartHealthCheck() {
	// Ticker będzie "tykał" co 5 sekund
	ticker := time.NewTicker(5 * time.Second)

	// Uruchamiamy osobną goroutine (wątek w tle)
	go func() {
		for range ticker.C {
			for _, b := range lb.backends {
				// Ustalamy krótki timeout, żeby nie czekać w nieskończoność na martwy serwer
				client := http.Client{Timeout: 2 * time.Second}

				resp, err := client.Get(b.URL.String())

				if err != nil || resp.StatusCode != http.StatusOK {
					if b.IsAlive() { // Zmiana stanu z żywego na martwy
						b.SetAlive(false)
						utils.Error("Backend %s PRZESTAŁ DZIAŁAĆ!", b.URL.String())
					}
				} else {
					if !b.IsAlive() { // Zmiana stanu z martwego na żywy
						b.SetAlive(true)
						utils.Info("Backend %s URUCHOMIONY PONOWNIE!", b.URL.String())
					}
					resp.Body.Close() // Zawsze zamykamy ciało odpowiedzi
				}
			}
		}
	}()
}
