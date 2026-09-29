package sshconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadResolvesHostsLikeSSH(t *testing.T) {
	dir := t.TempDir()
	extra := write(t, dir, "extra.conf", "Host nas\n  HostName nas.lan\n  Port 2222\n")
	main := write(t, dir, "config", `
Include `+extra+`

Host web
  HostName web.example.com
  User deploy
  IdentityFile ~/.ssh/web_key

Host bare
Host prod-* !prod-old
  User ops
Host *
  User fallback
  Port 2200
`)

	got, err := Load(main)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]struct {
		host, user, key string
		port            int
	}{}
	for _, c := range got {
		if c.Protocol != "SFTP" {
			t.Errorf("%s: protocol %s", c.Name, c.Protocol)
		}
		byName[c.Name] = struct {
			host, user, key string
			port            int
		}{c.Host, c.User, c.IdentityFile, c.Port}
	}

	want := map[string]struct {
		host, user, key string
		port            int
	}{
		"web":  {"web.example.com", "deploy", "~/.ssh/web_key", 2200},
		"bare": {"bare", "fallback", "", 2200},
		"nas":  {"nas.lan", "fallback", "", 2222},
	}
	if len(byName) != len(want) {
		t.Fatalf("got %d entries %v, want the 3 concrete aliases (wildcards and negations are not servers)", len(byName), byName)
	}
	for name, w := range want {
		if byName[name] != w {
			t.Errorf("%s = %+v, want %+v", name, byName[name], w)
		}
	}
}

func TestMissingFileIsNotAnError(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "nope"))
	if err != nil || got != nil {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestCircularIncludeIsAnErrorNotAHang(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config")
	write(t, dir, "config", "Include "+p+"\nHost a\n")
	if _, err := Load(p); err == nil {
		t.Fatal("a file that includes itself loaded without an error")
	}
}
