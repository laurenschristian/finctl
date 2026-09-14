// Package config resolves finctl settings: flags > env > config file. Provider
// API keys never live in plain text; a *_cmd prints them (Keychain, pass, op).
package config

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	UserAgent string `yaml:"user_agent,omitempty"` // sent to SEC etc.; should carry a contact
	IBKRURL   string `yaml:"ibkr_url,omitempty"`   // Client Portal gateway (ibkrctl daemon)
	VaultDir  string `yaml:"vault_dir,omitempty"`  // Obsidian vault root (read-only)
	CacheDir  string `yaml:"cache_dir,omitempty"`  // overrides the default cache location
	// Provider key commands: run to print a free API key. Env FINCTL_<NAME>_KEY overrides.
	FredKeyCmd    string `yaml:"fred_key_cmd,omitempty"`
	EIAKeyCmd     string `yaml:"eia_key_cmd,omitempty"`
	BLSKeyCmd     string `yaml:"bls_key_cmd,omitempty"`
	FinnhubKeyCmd string `yaml:"finnhub_key_cmd,omitempty"`
}

func supportDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "finctl")
}

func Path() string {
	if p := os.Getenv("FINCTL_CONFIG"); p != "" {
		return p
	}
	if p := os.Getenv("FIN_CONFIG"); p != "" {
		return p
	}
	return filepath.Join(supportDir(), "config.yaml")
}

func Load() (*Config, error) {
	c := &Config{}
	if b, err := os.ReadFile(Path()); err == nil {
		if err := yaml.Unmarshal(b, c); err != nil {
			return nil, err
		}
	}
	if v := os.Getenv("FINCTL_IBKR_URL"); v != "" {
		c.IBKRURL = v
	}
	if v := os.Getenv("FINCTL_VAULT_DIR"); v != "" {
		c.VaultDir = v
	}
	if v := os.Getenv("FINCTL_CACHE_DIR"); v != "" {
		c.CacheDir = v
	}
	if v := os.Getenv("FINCTL_USER_AGENT"); v != "" {
		c.UserAgent = v
	}
	if c.IBKRURL == "" {
		c.IBKRURL = "https://localhost:5001"
	}
	c.IBKRURL = strings.TrimRight(c.IBKRURL, "/")
	return c, nil
}

func Save(c *Config) error {
	p := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}

// UA returns the configured User-Agent, or a sensible default with a project
// contact (SEC requires a real, identifying UA).
func (c *Config) UA() string {
	if c.UserAgent != "" {
		return c.UserAgent
	}
	return "finctl (+https://github.com/laurenschristian/finctl)"
}

// CacheFile is the SQLite path.
func (c *Config) CacheFile() string {
	if c.CacheDir != "" {
		return filepath.Join(c.CacheDir, "cache.db")
	}
	return filepath.Join(supportDir(), "cache.db")
}

// Key resolves a provider API key: env FINCTL_<NAME>_KEY first, then the config
// *_cmd. Returns "" (no error) when neither is set, so callers can degrade.
func (c *Config) Key(provider string) (string, error) {
	if v := os.Getenv("FINCTL_" + strings.ToUpper(provider) + "_KEY"); v != "" {
		return v, nil
	}
	var cmd string
	switch provider {
	case "fred":
		cmd = c.FredKeyCmd
	case "eia":
		cmd = c.EIAKeyCmd
	case "bls":
		cmd = c.BLSKeyCmd
	case "finnhub":
		cmd = c.FinnhubKeyCmd
	}
	if cmd == "" {
		return "", nil
	}
	out, err := exec.CommandContext(context.Background(), "sh", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// Configured reports which known providers have a key available.
func (c *Config) Configured() map[string]bool {
	out := map[string]bool{}
	for _, p := range []string{"fred", "eia", "bls", "finnhub"} {
		k, _ := c.Key(p)
		out[p] = k != ""
	}
	return out
}
