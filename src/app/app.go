package app

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"

	"github.com/mslotwinski-dev/dash/src/config"
	"github.com/mslotwinski-dev/dash/src/middlewares"
	"github.com/mslotwinski-dev/dash/src/services"
	"github.com/mslotwinski-dev/dash/src/utils"
)

func Run() {
	cfg := config.LoadConfig()
	dashPath := utils.MakePath()

	backends := buildBackends(cfg.Backends)
	lb := &LoadBalancer{backends: backends}
	lb.StartHealthCheck()

	apiCache := NewCache(cfg.CacheTTL)
	mainRouter := newMainRouter(dashPath, lb, apiCache)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", mainRouter)

	limiter := middlewares.NewIPLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst)
	finalHandler := middlewares.RateLimitMiddleware(limiter, services.MetricsMiddleware(mux))

	certManager := newCertManager(dashPath, cfg.AutocertHosts)
	tlsServer := newTLSServer(finalHandler, &certManager, cfg.HTTPSPort)

	startHTTPRedirect(&certManager, cfg.HTTPPort)
	startHTTPSServer(tlsServer, cfg.HTTPSPort)
}
