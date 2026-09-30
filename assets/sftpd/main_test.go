package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// The one piece of real logic here: every incoming SFTP path must resolve
// under root, even a client attempting to walk out of it with "..". Expected
// paths are built with filepath.Join too, not hardcoded with "/" -- real()
// joins onto root with the host's own separator (root is a real filesystem
// path), so a literal "/srv/demo/..." string never matches on Windows.
func TestRealStaysInsideRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "srv", "demo")
	fs := rootedFS{root: root}

	cases := []struct{ in, want string }{
		{"/", root},
		{"/index.html", filepath.Join(root, "index.html")},
		{"/../../etc/passwd", filepath.Join(root, "etc", "passwd")},
		{"/a/../../../../secret", filepath.Join(root, "secret")},
	}
	for _, c := range cases {
		if got := fs.real(c.in); got != c.want {
			t.Errorf("real(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A restarted server has to be the same server to a client that already trusts
// it, or the recording of a dropped session would end in "host key changed".
func TestHostKeyPersistsOnlyWhenAskedTo(t *testing.T) {
	file := filepath.Join(t.TempDir(), "host_key")
	first, err := loadOrCreateHostKey(file)
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadOrCreateHostKey(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(first.PublicKey().Marshal()) != string(second.PublicKey().Marshal()) {
		t.Error("a second start with the same file made a different key")
	}

	a, _ := loadOrCreateHostKey("")
	b, _ := loadOrCreateHostKey("")
	if string(a.PublicKey().Marshal()) == string(b.PublicKey().Marshal()) {
		t.Error("without a file the key should be new every time")
	}
}

func TestAuthorizedKeysAreReadFromTheFile(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	key, _ := ssh.NewPublicKey(pub)
	other, _, _ := ed25519.GenerateKey(rand.Reader)
	otherKey, _ := ssh.NewPublicKey(other)

	file := filepath.Join(t.TempDir(), "authorized_keys")
	os.WriteFile(file, ssh.MarshalAuthorizedKey(key), 0o600)

	keys, err := loadAuthorizedKeys(file)
	if err != nil {
		t.Fatal(err)
	}
	if !keys[string(key.Marshal())] || keys[string(otherKey.Marshal())] {
		t.Errorf("membership wrong: %v", keys)
	}

	os.WriteFile(file, []byte("not a key\n"), 0o600)
	if _, err := loadAuthorizedKeys(file); err == nil {
		t.Error("a file with no keys was accepted")
	}
}

func TestDropAfterClosesTheConnectionOnSchedule(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()
	dropAfter(server, 30*time.Millisecond)

	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := client.Read(make([]byte, 1)); err == nil {
		t.Fatal("read succeeded on a connection that should have been cut")
	} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
		t.Fatal("the connection was never closed")
	}
}
