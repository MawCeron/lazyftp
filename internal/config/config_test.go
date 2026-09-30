package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

var conn = Connection{Host: "example.com", User: "ana", Port: 22, Protocol: "SFTP"}

func TestLoadMissingIsDefaults(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil || len(c.Connections) != 0 {
		t.Fatalf("got %v, %v", c, err)
	}
}

func TestLoadMalformedIsDefaultsWithError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(p, []byte("connections = [[["), 0o600)
	c, err := Load(p)
	if err == nil || len(c.Connections) != 0 {
		t.Fatalf("got %v, %v", c, err)
	}
}

func TestSaveLoadRoundTripHasNoPassword(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.toml")
	if err := Save(p, Config{Connections: []Connection{conn}}); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil || len(c.Connections) != 1 || c.Connections[0] != conn {
		t.Fatalf("got %v, %v", c, err)
	}
	raw, _ := os.ReadFile(p)
	if strings.Contains(strings.ToLower(string(raw)), "pass") {
		t.Fatalf("password field on disk:\n%s", raw)
	}
}

func TestSecretRoundTrip(t *testing.T) {
	keyring.MockInit()
	if _, err := Secret(conn); err != ErrNoSecret {
		t.Fatalf("want ErrNoSecret, got %v", err)
	}
	SetSecret(conn, "hunter2")
	if s, _ := Secret(conn); s != "hunter2" {
		t.Fatalf("got %q", s)
	}
	DeleteSecret(conn)
	if _, err := Secret(conn); err != ErrNoSecret {
		t.Fatalf("want ErrNoSecret after delete, got %v", err)
	}
}

func TestPushIsMostRecentFirstWithoutDuplicates(t *testing.T) {
	a, b, c := conn, conn, conn
	a.Host, b.Host, c.Host = "a", "b", "c"
	list := Push(nil, a, 2)
	list = Push(list, b, 2)
	list = Push(list, a, 2)
	if len(list) != 2 || list[0].Host != "a" || list[1].Host != "b" {
		t.Fatalf("got %v", list)
	}
	if list = Push(list, c, 2); len(list) != 2 || list[1].Host != "a" {
		t.Fatalf("oldest not dropped: %v", list)
	}
}

// os.UserConfigDir reads a different variable on each platform.
func isolateConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, k := range []string{"XDG_CONFIG_HOME", "APPDATA", "HOME", "USERPROFILE"} {
		t.Setenv(k, dir)
	}
	return dir
}

func TestThemeKeyRoundTripsAndIsOmittedWhenUnset(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	Save(p, Config{Theme: "borland", Connections: []Connection{conn}})
	if c, err := Load(p); err != nil || c.Theme != "borland" {
		t.Fatalf("%+v, %v", c, err)
	}
	Save(p, Config{})
	if raw, _ := os.ReadFile(p); strings.Contains(string(raw), "theme") {
		t.Fatalf("an unset theme was written:\n%s", raw)
	}
}

func TestThemeNamesCannotEscapeTheThemesDirectory(t *testing.T) {
	isolateConfigDir(t)
	for _, bad := range []string{"", "../config", "a/b", `a\b`, ".hidden", "x y", "a.toml/../../b"} {
		if _, err := ThemePath(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	p, err := ThemePath("borland-2.1")
	if err != nil || filepath.Base(p) != "borland-2.1.toml" || filepath.Base(filepath.Dir(p)) != "themes" {
		t.Fatalf("%q, %v", p, err)
	}
}

func TestLoadThemeReportsEachFailureInWords(t *testing.T) {
	isolateConfigDir(t)

	if _, err := LoadTheme("nope"); err == nil || !strings.Contains(err.Error(), "not found") || !strings.Contains(err.Error(), "nope.toml") {
		t.Errorf("missing: %v", err)
	}

	path, _ := ThemePath("good")
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("[dark]\nbackground = \"#0000AA\"\n"), 0o644)
	if th, err := LoadTheme("good"); err != nil || th.Dark.Background != "#0000AA" {
		t.Errorf("good: %+v, %v", th, err)
	}

	bad, _ := ThemePath("bad")
	os.WriteFile(bad, []byte("[dark]\naccent = \"green\"\n"), 0o644)
	if _, err := LoadTheme("bad"); err == nil || !strings.Contains(err.Error(), "accent") {
		t.Errorf("invalid: %v", err)
	}

	if _, err := LoadTheme("../x"); err == nil {
		t.Error("a path in the name was accepted")
	}
}
