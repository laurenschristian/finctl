package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadAndDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	t.Setenv("FINCTL_CONFIG", p)
	if err := Save(&Config{VaultDir: "/v", FredKeyCmd: "echo k"}); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(p); st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", st.Mode())
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.IBKRURL != "https://localhost:5001" || c.VaultDir != "/v" {
		t.Fatalf("defaults %+v", c)
	}
	if c.UA() == "" || !strings.Contains(c.UA(), "finctl") {
		t.Fatalf("ua %q", c.UA())
	}
}

func TestKeyResolution(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	t.Setenv("FINCTL_CONFIG", p)
	_ = Save(&Config{FredKeyCmd: "echo fromcmd"})
	c, _ := Load()
	if k, _ := c.Key("fred"); k != "fromcmd" {
		t.Fatalf("cmd key %q", k)
	}
	t.Setenv("FINCTL_FRED_KEY", "fromenv")
	if k, _ := c.Key("fred"); k != "fromenv" {
		t.Fatalf("env key %q", k)
	}
	if k, _ := c.Key("eia"); k != "" {
		t.Fatalf("unset key %q", k)
	}
	if !c.Configured()["fred"] || c.Configured()["bls"] {
		t.Fatalf("configured %+v", c.Configured())
	}
}

func TestPathAndBadYAML(t *testing.T) {
	t.Setenv("FINCTL_CONFIG", "")
	t.Setenv("FIN_CONFIG", "")
	if !strings.HasSuffix(Path(), filepath.Join("finctl", "config.yaml")) {
		t.Fatal(Path())
	}
	p := filepath.Join(t.TempDir(), "c.yaml")
	t.Setenv("FINCTL_CONFIG", p)
	_ = os.WriteFile(p, []byte("vault_dir: [x"), 0o600)
	if _, err := Load(); err == nil {
		t.Fatal("want yaml error")
	}
}
