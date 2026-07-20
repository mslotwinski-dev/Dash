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
	services.InitAccessLog()
	defer services.CloseAccessLog()

	cfg := config.LoadConfig()
	dashPath := utils.MakePath()

	re := NewRouteEngine(cfg.Routes)

	config.OnConfigChange = append(config.OnConfigChange, func(newCfg *config.Config) {
		utils.Info("Aktualizuję trasy RouteEngine w locie...")
		re.UpdateRoutes(newCfg.Routes)
	})

	hub := services.NewWsHub()
	hub.Start()

	apiCache := services.NewCache(cfg.Global.GetCacheTTL())
	mainRouter := newMainRouter(dashPath, re, apiCache)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWS(w, r, re.GetAllBackends)
	})

	mux.HandleFunc("/dash/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWS(w, r, re.GetAllBackends)
	})

	mux.HandleFunc("/dash/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(services.DashboardHTML))
	})

	mux.HandleFunc("/dash/api/backends", func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Query().Get("target")
		if target == "" {
			http.Error(w, "Brak parametru target", http.StatusBadRequest)
			return
		}
		if r.Method == http.MethodPost {
			backends := backend.BuildBackends([]string{target})
			if len(backends) > 0 {
				if lb := re.GetFirstLoadBalancer(); lb != nil {
					lb.AddBackend(backends[0])
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("Dodano backend: " + target))
				} else {
					http.Error(w, "Brak aktywnej trasy", http.StatusInternalServerError)
				}
			} else {
				http.Error(w, "Błąd budowy backendu", http.StatusInternalServerError)
			}
		} else if r.Method == http.MethodDelete {
			if lb := re.GetFirstLoadBalancer(); lb != nil {
				lb.RemoveBackend(target)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Usunięto backend: " + target))
			}
		} else {
			http.Error(w, "Metoda niedozwolona", http.StatusMethodNotAllowed)
		}
	})

	mux.Handle("/", mainRouter)

	limiter := middlewares.NewIPLimiter(rate.Limit(cfg.Global.RateLimitRPS), cfg.Global.RateLimitBurst)
	metricsHandler := services.MetricsMiddleware(hub, re.GetAllBackends)(mux)
	limitHandler := middlewares.RateLimitMiddleware(limiter, metricsHandler)
	securityHandler := middlewares.SecurityMiddleware(cfg.Security.Whitelist, cfg.Security.Blacklist)(limitHandler)
	finalHandler := middlewares.GzipMiddleware(securityHandler)

	if hasPublicAutocertHost(cfg.Global.AutocertHosts) && cfg.Global.EnableHTTPS {
		certManager := newCertManager(dashPath, cfg.Global.AutocertHosts)
		tlsServer := newTLSServer(finalHandler, &certManager, cfg.Global.HTTPSPort)

		if cfg.Global.RedirectToHTTPS {
			startHTTPRedirect(&certManager, cfg.Global.HTTPPort)
		}
		startHTTPSServer(tlsServer, cfg.Global.HTTPSPort)
		return
	}

	if cfg.Global.LocalHTTPS {
		utils.Warn("Brak publicznych hostów dla Autocert. Uruchamiam lokalne HTTP i HTTPS na portach %s / %s.", cfg.Global.HTTPPort, cfg.Global.HTTPSPort)
	} else {
		utils.Warn("Brak publicznych hostów dla Autocert. Uruchamiam tylko lokalne HTTP na porcie %s.", cfg.Global.HTTPPort)
	}

	startHTTPServer(finalHandler, cfg.Global.HTTPPort)

	if cfg.Global.LocalHTTPS {
		localTLSServer := newLocalTLSServer(finalHandler, cfg.Global.HTTPSPort)
		startHTTPSServer(localTLSServer, cfg.Global.HTTPSPort)
	}
}
