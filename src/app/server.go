package app

import (
	"net/http"
	"path/filepath"

	"github.com/mslotwinski-dev/dash/src/utils"
	"golang.org/x/crypto/acme/autocert"
)

func newCertManager(dashPath string, hosts []string) autocert.Manager {
	return autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(hosts...),
		Cache:      autocert.DirCache(filepath.Join(dashPath, "certs")),
	}
}

func newTLSServer(handler http.Handler, certManager *autocert.Manager, httpsPort string) *http.Server {
	return &http.Server{
		Addr:      httpsPort,
		Handler:   handler,
		TLSConfig: certManager.TLSConfig(),
	}
}

func startHTTPRedirect(certManager *autocert.Manager, httpPort string) {
	go func() {
		utils.Info("Uruchamiam przekierowanie HTTP -> HTTPS na porcie %s", httpPort)
		err := http.ListenAndServe(httpPort, certManager.HTTPHandler(nil))
		if err != nil {
			utils.Warn("Nie można uruchomić serwera na porcie %s (prawdopodobnie brak uprawnień administratora): %v", httpPort, err)
		}
	}()
}

func startHTTPSServer(server *http.Server, httpsPort string) {
	utils.Info("Serwer Dash gotowy na bezpieczne połączenia HTTPS na porcie %s!", httpsPort)
	err := server.ListenAndServeTLS("", "")
	if err != nil {
		utils.Critical("Krytyczny błąd serwera HTTPS: %v", err)
	}
}
