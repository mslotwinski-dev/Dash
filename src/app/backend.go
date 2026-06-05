package app

import (
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/mslotwinski-dev/dash/src/utils"
)

type Backend struct {
	URL          *url.URL
	ReverseProxy *httputil.ReverseProxy
	alive        bool
	mux          sync.RWMutex // Chroni dostęp do zmiennej alive
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.alive = alive
	b.mux.Unlock()
}

func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	alive := b.alive
	b.mux.RUnlock()
	return alive
}

func buildBackends(targets []string) []*Backend {
	backends := make([]*Backend, 0, len(targets))

	for _, target := range targets {
		u, err := url.Parse(target)
		if err != nil {
			utils.Error("Nie można odczytać backendu %s: %v", target, err)
			continue
		}

		backend := &Backend{
			URL:          u,
			ReverseProxy: httputil.NewSingleHostReverseProxy(u),
			alive:        true,
		}
		backends = append(backends, backend)
		utils.Info("Zarejestrowano backend: %s", target)
	}

	return backends
}
