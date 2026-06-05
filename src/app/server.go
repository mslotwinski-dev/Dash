package app

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mslotwinski-dev/dash/src/utils"
	"golang.org/x/crypto/acme/autocert"
)

var (
	selfCertsMu sync.Mutex
	selfCerts   = map[string]*tls.Certificate{}
)

func newCertManager(dashPath string, hosts []string) autocert.Manager {
	allowed := filterAutocertHosts(hosts)

	return autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(allowed...),
		Cache:      autocert.DirCache(filepath.Join(dashPath, "certs")),
	}
}

func hasPublicAutocertHost(hosts []string) bool {
	return len(filterAutocertHosts(hosts)) > 0
}

func filterAutocertHosts(hosts []string) []string {
	allowed := []string{}
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if h == "" || h == "localhost" {
			continue
		}
		if net.ParseIP(h) != nil {
			continue
		}
		if len(h) > 0 && h[0] == '[' {
			continue
		}
		if !strings.Contains(h, ".") {
			continue
		}
		allowed = append(allowed, h)
	}
	return allowed
}

func newTLSServer(handler http.Handler, certManager *autocert.Manager, httpsPort string) *http.Server {
	tlsCfg := certManager.TLSConfig()

	origGet := tlsCfg.GetCertificate
	tlsCfg.GetCertificate = func(ci *tls.ClientHelloInfo) (*tls.Certificate, error) {
		// If client provided an IP or localhost as SNI, return/generate a self-signed cert
		if ci == nil || ci.ServerName == "" || ci.ServerName == "localhost" || net.ParseIP(ci.ServerName) != nil {
			name := ci.ServerName
			if name == "" {
				name = "localhost"
			}
			return getOrCreateSelfSigned(name)
		}
		return origGet(ci)
	}

	return &http.Server{
		Addr:      httpsPort,
		Handler:   handler,
		TLSConfig: tlsCfg,
	}
}

func newLocalTLSServer(handler http.Handler, httpsPort string) *http.Server {
	cert, err := generateLocalDevCertificate()
	if err != nil {
		utils.Critical("Nie można wygenerować lokalnego certyfikatu TLS: %v", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	return &http.Server{
		Addr:      httpsPort,
		Handler:   handler,
		TLSConfig: tlsCfg,
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

func startHTTPServer(handler http.Handler, httpPort string) {
	utils.Info("Serwer Dash gotowy na połączenia HTTP na porcie %s!", httpPort)
	err := http.ListenAndServe(httpPort, handler)
	if err != nil {
		utils.Critical("Krytyczny błąd serwera HTTP: %v", err)
	}
}

func startHTTPSServer(server *http.Server, httpsPort string) {
	utils.Info("Serwer Dash gotowy na bezpieczne połączenia HTTPS na porcie %s!", httpsPort)
	err := server.ListenAndServeTLS("", "")
	if err != nil {
		utils.Critical("Krytyczny błąd serwera HTTPS: %v", err)
	}
}

func getOrCreateSelfSigned(name string) (*tls.Certificate, error) {
	selfCertsMu.Lock()
	if c, ok := selfCerts[name]; ok {
		selfCertsMu.Unlock()
		return c, nil
	}
	selfCertsMu.Unlock()

	cert, err := generateSelfSigned(name)
	if err != nil {
		return nil, err
	}

	selfCertsMu.Lock()
	selfCerts[name] = cert
	selfCertsMu.Unlock()
	return cert, nil
}

func generateSelfSigned(host string) (*tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	ip := net.ParseIP(host)
	if ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{host}
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tlsCert, nil
}

func generateLocalDevCertificate() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return tls.X509KeyPair(certPEM, keyPEM)
}
