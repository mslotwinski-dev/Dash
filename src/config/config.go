package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime-configurable values for the Dash server.
type Config struct {
	Backends       []string
	AutocertHosts  []string
	CacheTTL       time.Duration
	RateLimitRPS   int
	RateLimitBurst int
	HTTPPort       string
	HTTPSPort      string
}

// LoadConfig loads configuration from environment variables with sensible defaults.
func LoadConfig() *Config {
	cfg := &Config{
		Backends:       []string{"http://localhost:3000", "http://localhost:3001"},
		AutocertHosts:  []string{"twoja-domena.pl", "www.twoja-domena.pl"},
		CacheTTL:       10 * time.Second,
		RateLimitRPS:   3,
		RateLimitBurst: 5,
		HTTPPort:       ":80",
		HTTPSPort:      ":443",
	}

	if v := os.Getenv("DASH_BACKENDS"); v != "" {
		cfg.Backends = strings.Split(v, ",")
	}
	if v := os.Getenv("DASH_AUTOCERT_HOSTS"); v != "" {
		cfg.AutocertHosts = strings.Split(v, ",")
	}
	if v := os.Getenv("DASH_CACHE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.CacheTTL = d
		}
	}
	if v := os.Getenv("DASH_RATE_RPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimitRPS = n
		}
	}
	if v := os.Getenv("DASH_RATE_BURST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimitBurst = n
		}
	}
	if v := os.Getenv("DASH_HTTP_PORT"); v != "" {
		cfg.HTTPPort = v
	}
	if v := os.Getenv("DASH_HTTPS_PORT"); v != "" {
		cfg.HTTPSPort = v
	}

	return cfg
}
