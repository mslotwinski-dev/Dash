package app

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"

	"github.com/mslotwinski-dev/dash/src/backend"
	"github.com/mslotwinski-dev/dash/src/config"
	"github.com/mslotwinski-dev/dash/src/middlewares"
	"github.com/mslotwinski-dev/dash/src/services"
	"github.com/mslotwinski-dev/dash/src/utils"
)

func Run() {
	cfg := config.LoadConfig()
	dashPath := utils.MakePath()

	backends := backend.BuildBackends(cfg.Backends)
	lb := middlewares.NewLoadBalancer(backends)
	lb.StartHealthCheck()

	hub := services.NewWsHub()
	hub.Start()

	apiCache := services.NewCache(cfg.CacheTTL)
	mainRouter := newMainRouter(dashPath, lb, apiCache)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWS(w, r, lb)
	})

	mux.HandleFunc("/dash/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWS(w, r, lb)
	})

	mux.HandleFunc("/dash/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(services.DashboardHTML))
	})

	mux.Handle("/", mainRouter)

	limiter := middlewares.NewIPLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst)

	metricsHandler := services.MetricsMiddleware(hub, lb)(mux)

	finalHandler := middlewares.RateLimitMiddleware(limiter, metricsHandler)

	if hasPublicAutocertHost(cfg.AutocertHosts) {
		certManager := newCertManager(dashPath, cfg.AutocertHosts)
		tlsServer := newTLSServer(finalHandler, &certManager, cfg.HTTPSPort)

		if cfg.RedirectToHTTPS {
			startHTTPRedirect(&certManager, cfg.HTTPPort)
		}
		startHTTPSServer(tlsServer, cfg.HTTPSPort)
		return
	}

	if cfg.LocalHTTPS {
		utils.Warn("Brak publicznych hostów dla Autocert. Uruchamiam lokalne HTTP i HTTPS na portach %s / %s.", cfg.HTTPPort, cfg.HTTPSPort)
	} else {
		utils.Warn("Brak publicznych hostów dla Autocert. Uruchamiam tylko lokalne HTTP na porcie %s.", cfg.HTTPPort)
	}
	go startHTTPServer(finalHandler, cfg.HTTPPort)
	if cfg.LocalHTTPS {
		localTLSServer := newLocalTLSServer(finalHandler, cfg.HTTPSPort)
		startHTTPSServer(localTLSServer, cfg.HTTPSPort)
	}
}
