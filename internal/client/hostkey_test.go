package client

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func newKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	k, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestHostKeyCallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".ssh", "known_hosts")
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	remote := &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 22}
	host := "example.com:22"
	key := newKey(t)

	prompts := 0
	answer := true
	cb, err := hostKeyCallback(path, func(string, string, string) bool { prompts++; return answer }, a)
	if err != nil {
		t.Fatal(err)
	}

	answer = false
	if err := cb(host, remote, key); err == nil || prompts != 1 {
		t.Fatalf("rejected host: err=%v prompts=%d", err, prompts)
	}

	answer = true
	if err := cb(host, remote, key); err != nil || prompts != 2 {
		t.Fatalf("accepted host: err=%v prompts=%d", err, prompts)
	}

	// A new callback reads what the last one recorded: no prompt this time.
	cb, _ = hostKeyCallback(path, func(string, string, string) bool { prompts++; return true }, a)
	if err := cb(host, remote, key); err != nil || prompts != 2 {
		t.Fatalf("known host: err=%v prompts=%d", err, prompts)
	}

	if err := cb(host, remote, newKey(t)); err == nil || !strings.Contains(err.Error(), "changed") || prompts != 2 {
		t.Fatalf("changed key: err=%v prompts=%d", err, prompts)
	}

	cb, _ = hostKeyCallback(filepath.Join(t.TempDir(), "kh"), nil, a)
	if err := cb(host, remote, key); err == nil {
		t.Fatal("unknown host with no prompt was trusted")
	}
}
