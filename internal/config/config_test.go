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
