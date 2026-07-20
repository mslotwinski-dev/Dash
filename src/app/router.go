package app

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mslotwinski-dev/dash/src/middlewares"
	"github.com/mslotwinski-dev/dash/src/services"
	"github.com/mslotwinski-dev/dash/src/utils"
)

func newMainRouter(dashPath string, re *RouteEngine, apiCache *services.Cache) http.Handler {
	fileServer := http.FileServer(http.Dir(dashPath))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		matchedRoute := re.Match(r)

		if matchedRoute != nil && matchedRoute.LB != nil {
			if matchedRoute.Limiter != nil && !matchedRoute.Limiter.GetLimiter(r.RemoteAddr).Allow() {
				utils.Warn("[RATE LIMIT] Zbyt dużo zapytań z IP: %s (Trasa: %s)", r.RemoteAddr, matchedRoute.ID)
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			handleRoute(w, r, matchedRoute.LB, apiCache)
			return
		}

		if serveStaticAsset(w, r, dashPath, fileServer) {
			return
		}

		utils.Warn("[404 NOT FOUND] Brak dopasowanej trasy lub pliku: %s (Host: %s)", r.URL.Path, r.Host)
		http.NotFound(w, r)
	})
}

func handleRoute(w http.ResponseWriter, r *http.Request, lb *middlewares.LoadBalancer, apiCache *services.Cache) {
	cacheKey := r.Method + ":" + r.URL.Path + "?" + r.URL.RawQuery

	if r.Method == http.MethodGet && !strings.Contains(r.Header.Get("Cache-Control"), "no-cache") {
		if cachedResponse, found := apiCache.Get(cacheKey); found {
			if match := r.Header.Get("If-None-Match"); match != "" && match == cachedResponse.Header.Get("ETag") {
				w.WriteHeader(http.StatusNotModified)
				return
			}
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

	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
	}

	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		nextBackend := lb.GetNextBackend(r)
		if nextBackend == nil {
			utils.Error("Wszystkie backendy leżą! Zwracam 503")
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}

		if len(bodyBytes) > 0 {
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		utils.Warn("[API MISS] Próba %d. Przekierowuję do backendu: %s%s", attempt+1, nextBackend.URL.String(), r.URL.Path)

		nextBackend.IncConnections()
		
		var statusCode int
		var isError bool
		
		if r.Method == http.MethodGet {
			crw := services.NewCRW(w)
			nextBackend.ReverseProxy.ServeHTTP(crw, r)
			statusCode = crw.GetStatusCode()
			isError = (statusCode == http.StatusBadGateway || statusCode == http.StatusServiceUnavailable || statusCode == http.StatusGatewayTimeout)

			if !isError && statusCode == http.StatusOK {
				header := crw.GetHeader()
				if header.Get("ETag") == "" {
					hash := md5.Sum(crw.GetBodyBuf())
					header.Set("ETag", fmt.Sprintf(`"%x"`, hash))
				}
				apiCache.Set(cacheKey, crw.GetStatusCode(), header, crw.GetBodyBuf())
				utils.Info("[CACHE SAVED] Zapisano odpowiedź dla %s w RAM-ie", r.URL.Path)
			}
		} else {
			rrw := &RetryResponseWriter{ResponseWriter: w}
			nextBackend.ReverseProxy.ServeHTTP(rrw, r)
			statusCode = rrw.statusCode
			isError = rrw.isError
		}

		nextBackend.DecConnections()

		if !isError {
			nextBackend.RecordSuccess()
			return
		}

		utils.Warn("[RETRY] Backend %s zwrócił %d, ponawiam (próba %d/%d)...", nextBackend.URL.String(), statusCode, attempt+1, maxRetries)
	}

	http.Error(w, "Bad Gateway (po 3 próbach)", http.StatusBadGateway)
}

type RetryResponseWriter struct {
	http.ResponseWriter
	statusCode int
	isError    bool
}

func (rw *RetryResponseWriter) WriteHeader(code int) {
	if code == http.StatusBadGateway || code == http.StatusServiceUnavailable || code == http.StatusGatewayTimeout {
		rw.isError = true
		rw.statusCode = code
		return
	}
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *RetryResponseWriter) Write(b []byte) (int, error) {
	if rw.isError {
		return len(b), nil
	}
	return rw.ResponseWriter.Write(b)
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
