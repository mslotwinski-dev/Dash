package app

import (
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/mslotwinski-dev/dash/src/backend"
	"github.com/mslotwinski-dev/dash/src/config"
	"github.com/mslotwinski-dev/dash/src/middlewares"
	"golang.org/x/time/rate"
)

type Route struct {
	ID         string
	Host       string
	PathPrefix string
	LB         *middlewares.LoadBalancer
	Limiter    *middlewares.IPLimiter
}

type RouteEngine struct {
	mu     sync.RWMutex
	routes []*Route
}

func NewRouteEngine(cfgRoutes []config.RouteConfig) *RouteEngine {
	re := &RouteEngine{}
	re.UpdateRoutes(cfgRoutes)
	return re
}

func (re *RouteEngine) UpdateRoutes(cfgRoutes []config.RouteConfig) {
	re.mu.Lock()
	defer re.mu.Unlock()

	var newRoutes []*Route
	for _, cr := range cfgRoutes {
		var targets []string
		for _, b := range cr.Backends {
			targets = append(targets, b.URL+"|weight="+strconv.Itoa(b.Weight))
		}

		backends := backend.BuildBackends(targets)
		lb := middlewares.NewLoadBalancer(backends, cr.Strategy)
		lb.StartHealthCheck()

		var limiter *middlewares.IPLimiter
		if cr.RateLimitRPS > 0 {
			limiter = middlewares.NewIPLimiter(rate.Limit(cr.RateLimitRPS), cr.RateLimitBurst)
		}

		newRoutes = append(newRoutes, &Route{
			ID:         cr.ID,
			Host:       cr.Host,
			PathPrefix: cr.PathPrefix,
			LB:         lb,
			Limiter:    limiter,
		})
	}
	re.routes = newRoutes
}

func (re *RouteEngine) Match(r *http.Request) *Route {
	re.mu.RLock()
	defer re.mu.RUnlock()

	for _, route := range re.routes {
		if route.Host != "" && route.Host != "*" && route.Host != r.Host {
			continue
		}
		if route.PathPrefix != "" && !strings.HasPrefix(r.URL.Path, route.PathPrefix) {
			continue
		}
		return route
	}
	return nil
}

func (re *RouteEngine) GetAllBackends() []*backend.Backend {
	re.mu.RLock()
	defer re.mu.RUnlock()
	var all []*backend.Backend
	for _, route := range re.routes {
		all = append(all, route.LB.GetBackends()...)
	}
	return all
}

func (re *RouteEngine) GetFirstLoadBalancer() *middlewares.LoadBalancer {
	re.mu.RLock()
	defer re.mu.RUnlock()
	if len(re.routes) > 0 {
		return re.routes[0].LB
	}
	return nil
}
