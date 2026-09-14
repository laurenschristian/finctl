package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadResolve(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	t.Setenv("FIN_CONFIG", p)
	for _, k := range []string{"FIN_URL", "FIN_USER", "FIN_PASS"} {
		t.Setenv(k, "")
	}
	if err := Save(&Config{URL: "http://a/", Username: "u", PasswordCmd: "echo secret"}); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", st.Mode())
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Resolve(); err != nil || c.Password != "secret" || c.URL != "http://a" {
		t.Fatalf("%v %+v", err, c)
	}
	t.Setenv("FIN_URL", "http://b")
	t.Setenv("FIN_USER", "v")
	t.Setenv("FIN_PASS", "pw")
	c, _ = Load()
	_ = c.Resolve()
	if c.URL != "http://b" || c.Username != "v" || c.Password != "pw" {
		t.Fatalf("env override %+v", c)
	}
}

func TestPathAndBadYAML(t *testing.T) {
	t.Setenv("FIN_CONFIG", "")
	if !strings.HasSuffix(Path(), filepath.Join("finctl", "config.yaml")) {
		t.Fatal(Path())
	}
	p := filepath.Join(t.TempDir(), "c.yaml")
	t.Setenv("FIN_CONFIG", p)
	_ = os.WriteFile(p, []byte("url: [x"), 0o600)
	if _, err := Load(); err == nil {
		t.Fatal("want yaml error")
	}
}
