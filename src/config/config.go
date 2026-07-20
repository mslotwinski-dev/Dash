package config

import (
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mslotwinski-dev/dash/src/utils"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Global   GlobalConfig   `yaml:"global"`
	Security SecurityConfig `yaml:"security"`
	Routes   []RouteConfig  `yaml:"routes"`
}

type GlobalConfig struct {
	HTTPPort        string   `yaml:"http_port"`
	HTTPSPort       string   `yaml:"https_port"`
	EnableHTTPS     bool     `yaml:"enable_https"`
	LocalHTTPS      bool     `yaml:"local_https"`
	RedirectToHTTPS bool     `yaml:"redirect_to_https"`
	AutocertHosts   []string `yaml:"autocert_hosts"`
	CacheTTL        string   `yaml:"cache_ttl"`
	RateLimitRPS    int      `yaml:"rate_limit_rps"`
	RateLimitBurst  int      `yaml:"rate_limit_burst"`
}

func (g GlobalConfig) GetCacheTTL() time.Duration {
	d, err := time.ParseDuration(g.CacheTTL)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

type SecurityConfig struct {
	Whitelist []string `yaml:"whitelist"`
	Blacklist []string `yaml:"blacklist"`
}

type RouteConfig struct {
	ID             string          `yaml:"id"`
	Host           string          `yaml:"host"`
	PathPrefix     string          `yaml:"path_prefix"`
	Strategy       string          `yaml:"strategy"`
	RateLimitRPS   int             `yaml:"rate_limit_rps"`
	RateLimitBurst int             `yaml:"rate_limit_burst"`
	Backends       []BackendConfig `yaml:"backends"`
}

type BackendConfig struct {
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight"`
}

var (
	CurrentConfig  *Config
	configPath     = "dash.yaml"
	OnConfigChange []func(*Config)
)

func LoadConfig() *Config {
	cfg := &Config{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		utils.Error("Błąd odczytu pliku %s: %v. Używam domyślnej konfiguracji.", configPath, err)
		cfg = getDefaultConfig()
	} else {
		err = yaml.Unmarshal(data, cfg)
		if err != nil {
			utils.Error("Błąd parsowania pliku YAML: %v. Używam domyślnej konfiguracji.", err)
			cfg = getDefaultConfig()
		}
	}

	CurrentConfig = cfg
	go watchConfig()
	return cfg
}

func getDefaultConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			HTTPPort:       ":80",
			HTTPSPort:      ":443",
			CacheTTL:       "10s",
			RateLimitRPS:   10,
			RateLimitBurst: 20,
		},
		Routes: []RouteConfig{
			{
				ID:         "default",
				PathPrefix: "/api",
				Strategy:   "round-robin",
			},
		},
	}
}

func watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		utils.Error("fsnotify NewWatcher error: %v", err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(configPath)
	if err != nil {
		utils.Error("fsnotify Add error: %v", err)
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) {
				utils.Info("Zauważono modyfikację pliku konfiguracyjnego. Przeładowywanie...")
				data, err := os.ReadFile(configPath)
				if err == nil {
					newCfg := &Config{}
					if err := yaml.Unmarshal(data, newCfg); err == nil {
						CurrentConfig = newCfg
						for _, fn := range OnConfigChange {
							fn(CurrentConfig)
						}
					} else {
						utils.Error("Błąd parsowania zaktualizowanego YAML: %v", err)
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			utils.Error("fsnotify event error: %v", err)
		}
	}
}
