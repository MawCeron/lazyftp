package ui

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/charmbracelet/bubbles/list"
)

// The bug this guards against: cursor and mark used to share one character
// slot, so selecting a marked file hid its checkmark.
func TestFileDelegateShowsCursorAndMarkTogether(t *testing.T) {
	items := []list.Item{fileItem{file: model.FileInfo{Name: "report.txt"}}}
	marked := map[int]bool{0: true}
	delegate := fileDelegate{marked: marked}
	l := list.New(items, delegate, 80, 10)
	l.Select(0)

	var buf bytes.Buffer
	delegate.Render(&buf, l, 0, items[0])
	out := buf.String()

	if !strings.Contains(out, ">") {
		t.Errorf("Render() = %q, want a cursor indicator", out)
	}
	if !strings.Contains(out, iconMark()) {
		t.Errorf("Render() = %q, want a mark indicator", out)
	}
}

func TestRemotePathsStayPOSIX(t *testing.T) {
	// Remote paths are POSIX regardless of the host lazyftp runs on, so these
	// expectations are literal on every platform.
	cases := []struct {
		name        string
		path, child string
		wantChild   string
		wantParent  string
	}{
		{name: "root", path: "/", child: "var", wantChild: "/var", wantParent: "/"},
		{name: "one level", path: "/var", child: "www", wantChild: "/var/www", wantParent: "/"},
		{name: "nested", path: "/var/www", child: "html", wantChild: "/var/www/html", wantParent: "/var"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Panel{path: c.path}
			if got := p.childPath(c.child); got != c.wantChild {
				t.Errorf("childPath(%q) = %q, want %q", c.child, got, c.wantChild)
			}
			if got := p.parentPath(); got != c.wantParent {
				t.Errorf("parentPath() = %q, want %q", got, c.wantParent)
			}
		})
	}
}

func TestRemotePathCollapsesDoubleSlashes(t *testing.T) {
	p := Panel{path: "/"}
	if got := p.cleanPath("//var//www//"); got != "/var/www" {
		t.Errorf("cleanPath = %q, want %q", got, "/var/www")
	}
}

// The reported crash: going up from the home directory on Windows. parentPath
// used to look for a forward slash, find none, and slice with a negative index.
func TestLocalParentPathOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows path semantics")
	}

	cases := map[string]string{
		`C:\Users\Mauricio`: `C:\Users`,
		`C:\Users`:          `C:\`,
		`C:\`:               `C:\`,
	}

	for in, want := range cases {
		p := Panel{path: in, local: true}
		if got := p.parentPath(); got != want {
			t.Errorf("parentPath(%q) = %q, want %q", in, got, want)
		}
	}
}

// Same walk on whatever host the tests run on, built with filepath so the
// expectations hold everywhere.
func TestLocalPathsFollowTheHost(t *testing.T) {
	base := filepath.Join(string(filepath.Separator), "home", "user")
	p := Panel{path: base, local: true}

	child := p.childPath("projects")
	if want := filepath.Join(base, "projects"); child != want {
		t.Errorf("childPath = %q, want %q", child, want)
	}

	if got := p.parentPath(); got != filepath.Dir(base) {
		t.Errorf("parentPath = %q, want %q", got, filepath.Dir(base))
	}
}
