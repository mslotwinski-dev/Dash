package src

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/mslotwinski-dev/dash/src/utils"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Run() {
	dashPath := MakePath()

	backendTargets := []string{
		"http://localhost:3000",
		"http://localhost:3001",
	}

	var backends []*Backend
	for _, target := range backendTargets {
		u, _ := url.Parse(target)
		b := &Backend{
			URL:          u,
			ReverseProxy: httputil.NewSingleHostReverseProxy(u),
			alive:        true,
		}
		backends = append(backends, b)
		utils.Info("Zarejestrowano backend: %s", target)
	}

	lb := &LoadBalancer{backends: backends}
	lb.StartHealthCheck()

	// 1. Tworzymy tradycyjny serwer plików
	baseFileServer := http.FileServer(http.Dir(dashPath))

	// 2. Owijamy TYLKO serwer plików w GzipMiddleware.
	// Dzięki temu API i błędy 404 są całkowicie bezpieczne i nigdy nie wywalą aplikacji!
	fileServer := GzipMiddleware(baseFileServer)

	apiCache := NewCache(10 * time.Second)

	mainRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. JAWNA REGUŁA DLA BACKENDU (API)
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {

			// --- TUTAJ ZACZYNA SIĘ CACHE ---
			cacheKey := r.Method + ":" + r.URL.Path + "?" + r.URL.RawQuery

			// Tylko zapytania GET powinny być cache'owane (POST/PUT/DELETE zmieniają dane!)
			if r.Method == http.MethodGet {
				if cachedResponse, found := apiCache.Get(cacheKey); found {
					utils.Info("[CACHE HIT] Zwracam dane z pamięci RAM dla: %s", r.URL.Path)

					// Przepisujemy nagłówki ze skrytki
					for k, vv := range cachedResponse.Header {
						for _, v := range vv {
							w.Header().Add(k, v)
						}
					}
					w.WriteHeader(cachedResponse.StatusCode)
					w.Write(cachedResponse.Body)
					return
				}
			}
			// --- KONIEC SPRAWDZANIA CACHE ---

			nextBackend := lb.GetNextBackend()
			if nextBackend == nil {
				utils.Error("Wszystkie backendy leżą! Zwracam 503")
				http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
				return
			}

			utils.Warn("[API MISS] Przekierowuję do backendu: %s%s", nextBackend.URL.String(), r.URL.Path)

			if r.Method == http.MethodGet {
				// Jeśli to GET, owijamy ResponseWriter, żeby przechwycić odpowiedź backendu do cache
				crw := &cacheResponseWriter{
					ResponseWriter: w,
					bodyBuf:        bytes.NewBuffer(nil),
					statusCode:     http.StatusOK, // Domyślnie 200 OK
				}

				nextBackend.ReverseProxy.ServeHTTP(crw, r)

				// Jeśli backend odpowiedział sukcesem (200), zapisujemy to do pamięci RAM!
				if crw.statusCode == http.StatusOK {
					apiCache.Set(cacheKey, crw.statusCode, crw.Header(), crw.bodyBuf.Bytes())
					utils.Info("[CACHE SAVED] Zapisano odpowiedź dla %s w RAM-ie", r.URL.Path)
				}
			} else {
				// Metody POST, PUT, DELETE lecą tradycyjnie, bez dotykania cache
				nextBackend.ReverseProxy.ServeHTTP(w, r)
			}
			return
		}

		if r.URL.Path == "/" {
			indexCheck := filepath.Join(dashPath, "index.html")
			if _, err := os.Stat(indexCheck); err == nil {
				utils.Info("[STATIC MODE] Serwuję główny plik index.html")
				fileServer.ServeHTTP(w, r) // Tutaj pójdzie przez Gzip
				return
			}
		}

		requestedFile := filepath.Join(dashPath, filepath.Clean(r.URL.Path))
		fileInfo, err := os.Stat(requestedFile)

		if err == nil && !fileInfo.IsDir() {
			utils.Info("[STATIC MODE] Znaleziono plik na dysku. Serwuję: %s", r.URL.Path)
			fileServer.ServeHTTP(w, r) // Tutaj pójdzie przez Gzip
			return
		}

		// Czysty błąd 404 - bez kombinowania z Gzipem, brak ryzyka paniki!
		utils.Warn("[404 NOT FOUND] Brak pliku na dysku i to nie jest API: %s", r.URL.Path)
		http.NotFound(w, r)
	})

	// Tworzymy główny Multiplexer (zarządcę ścieżek)
	mux := http.NewServeMux()

	// 1. JAWNY ENDPOINT DLA METRYK PROMETHEUSA
	// Pod tym adresem serwer będzie wypluwał dane liczbowe dla systemów monitoringu
	mux.Handle("/metrics", promhttp.Handler())

	// 2. CAŁA RESZTA RUCHU IDZIE PRZEZ NASZ FUNKCJONALNY ROUTER OWINIĘTY W METRYKI
	mux.Handle("/", MetricsMiddleware(mainRouter))

	utils.Info("Serwer Dash uruchomiony na porcie :8080!")
	utils.Info("Metryki dostępne pod: http://localhost:8080/metrics")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
