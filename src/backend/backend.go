package backend

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mslotwinski-dev/dash/src/utils"
)

type Backend struct {
	URL               *url.URL
	ReverseProxy      *httputil.ReverseProxy
	alive             bool
	Weight            int
	ActiveConnections int64
	Failures          int32
	CircuitOpenUntil  time.Time
	mux               sync.RWMutex // Chroni dostęp do zmiennej alive
}

func (b *Backend) IncConnections() {
	atomic.AddInt64(&b.ActiveConnections, 1)
}

func (b *Backend) DecConnections() {
	atomic.AddInt64(&b.ActiveConnections, -1)
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.alive = alive
	b.mux.Unlock()
}

func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	alive := b.alive
	openUntil := b.CircuitOpenUntil
	b.mux.RUnlock()
	
	if !alive && time.Now().After(openUntil) && !openUntil.IsZero() {
		// Circuit breaker timeout minął, spróbujmy ponownie puścić ruch
		b.SetAlive(true)
		atomic.StoreInt32(&b.Failures, 0)
		return true
	}
	
	return alive
}

func (b *Backend) RecordFailure() {
	fails := atomic.AddInt32(&b.Failures, 1)
	if fails >= 5 {
		b.mux.Lock()
		b.alive = false
		b.CircuitOpenUntil = time.Now().Add(15 * time.Second)
		b.mux.Unlock()
		utils.Warn("[CIRCUIT BREAKER] Odcięto backend %s na 15 sekund (5 błędów)", b.URL.String())
	}
}

func (b *Backend) RecordSuccess() {
	atomic.StoreInt32(&b.Failures, 0)
}

func BuildBackends(targets []string) []*Backend {
	backends := make([]*Backend, 0, len(targets))

	for _, target := range targets {
		rawUrl := target
		weight := 1
		
		if strings.Contains(target, "|weight=") {
			parts := strings.Split(target, "|weight=")
			rawUrl = parts[0]
			if w, err := strconv.Atoi(parts[1]); err == nil && w > 0 {
				weight = w
			}
		}

		u, err := url.Parse(rawUrl)
		if err != nil {
			utils.Error("Nie można odczytać backendu %s: %v", rawUrl, err)
			continue
		}

		backend := &Backend{
			URL:          u,
			ReverseProxy: httputil.NewSingleHostReverseProxy(u),
			alive:        true,
			Weight:       weight,
		}
		
		backend.ReverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			utils.Error("[PROXY ERROR] Błąd połączenia z %s: %v", backend.URL.String(), err)
			backend.RecordFailure()
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		}

		backends = append(backends, backend)
		utils.Info("Zarejestrowano backend: %s (Waga: %d)", rawUrl, weight)
	}

	return backends
}
