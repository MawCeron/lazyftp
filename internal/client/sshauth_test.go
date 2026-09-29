package client

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func writeKey(t *testing.T, home, name string, passphrase []byte) {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	var blk *pem.Block
	var err error
	if passphrase == nil {
		blk, err = ssh.MarshalPrivateKey(priv, "")
	} else {
		blk, err = ssh.MarshalPrivateKeyWithPassphrase(priv, "", passphrase)
	}
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(home, ".ssh"), 0o700)
	os.WriteFile(filepath.Join(home, ".ssh", name), pem.EncodeToMemory(blk), 0o600)
}

func TestSignersDefaultKeysAndPassphrase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SSH_AUTH_SOCK", "")
	writeKey(t, home, "id_ed25519", nil)
	writeKey(t, home, "id_rsa", []byte("s3cret"))

	a := &sshAuth{}
	s, _ := a.signers()
	if len(s) != 1 || len(a.locked) != 1 || a.locked[0] != "id_rsa" {
		t.Fatalf("no passphrase: %d signers, locked %v", len(s), a.locked)
	}
	if err := a.explain(os.ErrPermission); !strings.Contains(err.Error(), "id_rsa") {
		t.Fatalf("error does not name the locked key: %v", err)
	}

	a = &sshAuth{pass: "s3cret"}
	s, _ = a.signers()
	if len(s) != 2 || len(a.locked) != 0 || a.method != "public key (id_ed25519, id_rsa)" {
		t.Fatalf("with passphrase: %d signers, locked %v, method %q", len(s), a.locked, a.method)
	}
}

func TestPasswordMethodOnlyWhenGiven(t *testing.T) {
	if n := len((&sshAuth{}).methods()); n != 1 {
		t.Fatalf("got %d methods without a password, want 1", n)
	}
	if n := len((&sshAuth{pass: "x"}).methods()); n != 2 {
		t.Fatalf("got %d methods with a password, want 2", n)
	}
}

func TestIdentityFileAtAnyPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SSH_AUTH_SOCK", "")
	writeKey(t, home, "work_key", nil)
	writeKey(t, home, "locked_key", []byte("s3cret"))
	key := func(name string) string { return filepath.Join(home, ".ssh", name) }

	a := &sshAuth{identity: key("work_key")}
	if s, _ := a.signers(); len(s) != 1 || a.method != "public key (work_key)" || a.identityErr != nil {
		t.Fatalf("plain key: %d signers, method %q, err %v", len(s), a.method, a.identityErr)
	}

	a = &sshAuth{identity: "~/.ssh/work_key"}
	if s, _ := a.signers(); len(s) != 1 {
		t.Fatalf("~ not expanded: %d signers, err %v", len(s), a.identityErr)
	}

	a = &sshAuth{identity: key("locked_key"), pass: "s3cret"}
	if s, _ := a.signers(); len(s) != 1 {
		t.Fatalf("passphrase: %d signers, err %v", len(s), a.identityErr)
	}

	a = &sshAuth{identity: key("locked_key")}
	a.signers()
	if err := a.explain(os.ErrPermission); !strings.Contains(err.Error(), "passphrase") {
		t.Fatalf("locked key not explained: %v", err)
	}

	a = &sshAuth{identity: key("missing")}
	a.signers()
	if err := a.explain(os.ErrPermission); !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing key not named: %v", err)
	}
}
