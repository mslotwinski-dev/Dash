package app

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mslotwinski-dev/dash/src/middlewares"
	"github.com/mslotwinski-dev/dash/src/services"
	"github.com/mslotwinski-dev/dash/src/utils"
)

func newMainRouter(dashPath string, lb *middlewares.LoadBalancer, apiCache *services.Cache) http.Handler {
	fileServer := middlewares.GzipMiddleware(http.FileServer(http.Dir(dashPath)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			handleAPIRoute(w, r, lb, apiCache)
			return
		}

		if serveStaticAsset(w, r, dashPath, fileServer) {
			return
		}

		utils.Warn("[404 NOT FOUND] Brak pliku na dysku i to nie jest API: %s", r.URL.Path)
		http.NotFound(w, r)
	})
}

func handleAPIRoute(w http.ResponseWriter, r *http.Request, lb *middlewares.LoadBalancer, apiCache *services.Cache) {
	cacheKey := r.Method + ":" + r.URL.Path + "?" + r.URL.RawQuery

	if r.Method == http.MethodGet {
		if cachedResponse, found := apiCache.Get(cacheKey); found {
			utils.Info("[CACHE HIT] Zwracam dane z pamięci RAM dla: %s", r.URL.Path)

			for k, vv := range cachedResponse.Header {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(cachedResponse.StatusCode)
			_, _ = w.Write(cachedResponse.Body)
			return
		}
	}

	nextBackend := lb.GetNextBackend()
	if nextBackend == nil {
		utils.Error("Wszystkie backendy leżą! Zwracam 503")
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	utils.Warn("[API MISS] Przekierowuję do backendu: %s%s", nextBackend.URL.String(), r.URL.Path)

	if r.Method == http.MethodGet {
		crw := services.NewCRW(w)

		nextBackend.ReverseProxy.ServeHTTP(crw, r)

		if crw.GetStatusCode() == http.StatusOK {
			apiCache.Set(cacheKey, crw.GetStatusCode(), crw.GetHeader(), crw.GetBodyBuf())
			utils.Info("[CACHE SAVED] Zapisano odpowiedź dla %s w RAM-ie", r.URL.Path)
		}
		return
	}

	nextBackend.ReverseProxy.ServeHTTP(w, r)
}

func serveStaticAsset(w http.ResponseWriter, r *http.Request, dashPath string, fileServer http.Handler) bool {
	if r.URL.Path == "/" {
		indexCheck := filepath.Join(dashPath, "index.html")
		if _, err := os.Stat(indexCheck); err == nil {
			utils.Info("[STATIC MODE] Serwuję główny plik index.html")
			fileServer.ServeHTTP(w, r)
			return true
		}
	}

	requestedFile := filepath.Join(dashPath, filepath.Clean(r.URL.Path))
	fileInfo, err := os.Stat(requestedFile)
	if err == nil && !fileInfo.IsDir() {
		utils.Info("[STATIC MODE] Znaleziono plik na dysku. Serwuję: %s", r.URL.Path)
		fileServer.ServeHTTP(w, r)
		return true
	}

	return false
}
