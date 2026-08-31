package main

import "testing"

// The one piece of real logic here: every incoming SFTP path must resolve
// under root, even a client attempting to walk out of it with "..".
func TestRealStaysInsideRoot(t *testing.T) {
	fs := rootedFS{root: "/srv/demo"}

	cases := map[string]string{
		"/":                     "/srv/demo",
		"/index.html":           "/srv/demo/index.html",
		"/../../etc/passwd":     "/srv/demo/etc/passwd",
		"/a/../../../../secret": "/srv/demo/secret",
	}
	for in, want := range cases {
		if got := fs.real(in); got != want {
			t.Errorf("real(%q) = %q, want %q", in, got, want)
		}
	}
}
