package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	TTL              time.Duration            `yaml:"ttl"`
	TTLOverrides     map[string]time.Duration `yaml:"ttl_overrides"`
	MaxCacheEntries  int                      `yaml:"max_cache_entries"`
	SocketPath       string                   `yaml:"socket_path"`
	PIDFile          string                   `yaml:"pid_file"`
	AutoStart        bool                     `yaml:"auto_start"`
	AdditionalCache  []string                 `yaml:"additional_cacheable"`
	DashboardPort    int                      `yaml:"dashboard_port"`
	GHPath           string                   `yaml:"gh_path"`
	LogLevel         string                   `yaml:"log_level"`
	LogFile          string                   `yaml:"log_file"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	ghcDir := filepath.Join(home, ".ghc")
	return &Config{
		TTL:             30 * time.Second,
		TTLOverrides:    make(map[string]time.Duration),
		MaxCacheEntries: 1000,
		SocketPath:      filepath.Join(ghcDir, "ghcd.sock"),
		PIDFile:         filepath.Join(ghcDir, "ghcd.pid"),
		AutoStart:       true,
		DashboardPort:   9847,
		GHPath:          "gh",
		LogLevel:        "info",
		LogFile:         filepath.Join(ghcDir, "ghcd.log"),
	}
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil
	}

	configPath := filepath.Join(home, ".ghc", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return cfg, err
	}

	// Apply env var overrides
	if v := os.Getenv("GHC_TTL"); v != "" {
		if d, err := time.ParseDuration(v + "s"); err == nil {
			cfg.TTL = d
		} else if d, err := time.ParseDuration(v); err == nil {
			cfg.TTL = d
		}
	}
	if v := os.Getenv("GHC_SOCKET"); v != "" {
		cfg.SocketPath = v
	}
	if v := os.Getenv("GHC_GH_PATH"); v != "" {
		cfg.GHPath = v
	}

	return cfg, nil
}

// GHCDir returns the ghc configuration directory, creating it if needed.
func GHCDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".ghc")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// CommandTTL returns the TTL for a specific command, falling back to default.
func (c *Config) CommandTTL(cmd string) time.Duration {
	if ttl, ok := c.TTLOverrides[cmd]; ok {
		return ttl
	}
	return c.TTL
}
