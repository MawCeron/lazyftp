package main

import (
	"path/filepath"
	"testing"
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
