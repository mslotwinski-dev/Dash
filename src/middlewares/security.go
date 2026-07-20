package middlewares

import (
	"net"
	"net/http"
	"strings"

	"github.com/mslotwinski-dev/dash/src/utils"
)

func SecurityMiddleware(whitelist []string, blacklist []string) func(http.Handler) http.Handler {
	wlMap := make(map[string]bool)
	blMap := make(map[string]bool)

	for _, ip := range whitelist {
		if t := strings.TrimSpace(ip); t != "" {
			wlMap[t] = true
		}
	}
	for _, ip := range blacklist {
		if t := strings.TrimSpace(ip); t != "" {
			blMap[t] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)

			if len(wlMap) > 0 {
				if !wlMap[ip] {
					utils.Warn("[SECURITY] Zablokowano dostęp z IP: %s (Brak na Whitelist)", ip)
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}

			if len(blMap) > 0 {
				if blMap[ip] {
					utils.Warn("[SECURITY] Zablokowano dostęp z IP: %s (Obecny na Blacklist)", ip)
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
