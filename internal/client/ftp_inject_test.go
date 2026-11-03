package client

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func connectFakeFTP(t *testing.T, s *fakeFTP) *FTPClient {
	t.Helper()
	c := NewFTPClient(nil)
	if err := c.Connect("127.0.0.1", "u", "p", s.port()); err != nil {
		t.Fatalf("connecting to the fake server: %v", err)
	}
	t.Cleanup(func() { c.Disconnect() })
	return c
}

// A name that holds a line break becomes two FTP commands, and one that holds
// a NUL is cut short by some servers. Whatever a name came from, the client has
// to refuse it before anything is written to the connection.
func TestAFileNameThatCouldInjectACommandNeverReachesTheServer(t *testing.T) {
	hostile := []string{"a\r\nDELE injected", "a\nDELE injected", "a\rDELE injected", "a\x00b"}

	for _, name := range hostile {
		s := startFakeFTP(t)
		c := connectFakeFTP(t, s)
		before := len(s.sent())

		local := t.TempDir()
		localName := filepath.Join(local, name)
		hasLocalFile := os.WriteFile(localName, []byte("x"), 0o644) == nil

		ops := map[string]func() error{
			"Mkdir":           func() error { return c.Mkdir("/" + name) },
			"Rename (old)":    func() error { return c.Rename("/"+name, "/ok") },
			"Rename (new)":    func() error { return c.Rename("/ok", "/"+name) },
			"Delete a file":   func() error { return c.Delete("/"+name, false) },
			"Delete a folder": func() error { return c.Delete("/"+name, true) },
			"List":            func() error { _, err := c.List("/" + name); return err },
			"Download":        func() error { return c.Download(context.Background(), "/"+name, t.TempDir(), func(int64) {}) },
			"Upload to": func() error {
				return c.Upload(context.Background(), filepath.Join(t.TempDir(), "f"), "/"+name, func(int64) {})
			},
		}
		if hasLocalFile {
			ops["Upload of"] = func() error { return c.Upload(context.Background(), localName, "/", func(int64) {}) }
		}

		for op, run := range ops {
			if err := run(); err == nil {
				t.Errorf("%s with %q succeeded, want an error", op, name)
			}
			if s.tainted() || s.reached("DELE") {
				t.Fatalf("%s with %q put %q on the wire", op, name, s.sent()[before:])
			}
			if got := len(s.sent()); got != before {
				t.Fatalf("%s with %q sent %q, want nothing", op, name, s.sent()[before:])
			}
		}
	}
}

// A listing is the server's own text, so a name inside it can be hostile too,
// and deleting a folder recursively builds a command from every entry.
func TestARemoteListingEntryCannotInjectACommandWhenDeletingAFolder(t *testing.T) {
	s := startFakeFTP(t)
	s.listing = "-rw-r--r--   1 u g 5 Jan 01  2026 evil\rDELE injected\r\n"
	c := connectFakeFTP(t, s)

	if err := c.Delete("/folder", true); err == nil {
		t.Error("deleting a folder with a hostile entry succeeded, want an error")
	}
	if s.tainted() {
		t.Errorf("a hostile listing entry reached the wire: %q", s.sent())
	}
}

func TestOrdinaryNamesStillReachTheFTPServer(t *testing.T) {
	s := startFakeFTP(t)
	c := connectFakeFTP(t, s)

	if err := c.Mkdir("/my folder/niño (1).txt"); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := c.Rename("/my folder/niño (1).txt", "/日本語"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if err := c.Delete("/日本語", false); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	for _, want := range []string{"MKD /my folder/niño (1).txt", "RNFR /my folder/niño (1).txt", "RNTO /日本語", "DELE /日本語"} {
		found := false
		for _, l := range s.sent() {
			found = found || l == want
		}
		if !found {
			t.Errorf("the server never received %q; got %q", want, s.sent())
		}
	}
}
