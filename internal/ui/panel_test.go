package ui

import (
	"bytes"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/model"
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

// The bug this guards against: bubbles/list binds h/l to PrevPage/NextPage by
// default, and unhandled keys fall through to the list -- so h/l silently
// paginated instead of navigating.
func TestHAndLDoNotTriggerListPagination(t *testing.T) {
	files := make([]model.FileInfo, 30)
	for i := range files {
		files[i] = model.FileInfo{Name: fmt.Sprintf("file-%02d.txt", i)}
	}

	p := NewPanel("LOCAL", true).WithFiles(files, "/tmp")
	p = p.SetSize(40, 10) // short enough to force multiple pages
	before := p.list.Paginator.Page

	p, _ = p.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	if p.list.Paginator.Page != before {
		t.Errorf("l changed the page from %d to %d, want unchanged", before, p.list.Paginator.Page)
	}

	p, _ = p.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	if p.list.Paginator.Page != before {
		t.Errorf("h changed the page from %d to %d, want unchanged", before, p.list.Paginator.Page)
	}
}

func TestSpaceTogglesMark(t *testing.T) {
	files := []model.FileInfo{{Name: "a.txt"}, {Name: "b.txt"}}
	p := NewPanel("LOCAL", true).WithFiles(files, "/tmp")

	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !p.marked[0] {
		t.Fatal("space did not mark the file under the cursor")
	}

	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if p.marked[0] {
		t.Fatal("space did not unmark an already-marked file")
	}
}

func TestRefreshReloadsTheCurrentPath(t *testing.T) {
	p := NewPanel("REMOTE", false).WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/srv")

	_, cmd := p.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd == nil {
		t.Fatal("r did not return a command")
	}
	msg, ok := cmd().(NavigateMsg)
	if !ok {
		t.Fatalf("r returned %T, want NavigateMsg", cmd())
	}
	if msg.Panel != "REMOTE" || msg.Path != "/srv" {
		t.Errorf("r navigated to %+v, want {REMOTE /srv} (the current path)", msg)
	}
}

func TestFileDelegateColumnsDegradeWithWidth(t *testing.T) {
	modTime := time.Date(2026, 8, 15, 14, 30, 0, 0, time.UTC)
	file := fileItem{file: model.FileInfo{Name: "report.txt", Size: 2048, ModTime: modTime}}
	items := []list.Item{file}
	delegate := fileDelegate{marked: map[int]bool{}}

	render := func(width int) string {
		l := list.New(items, delegate, width, 10)
		var buf bytes.Buffer
		delegate.Render(&buf, l, 0, items[0])
		return buf.String()
	}

	wantDate := "2026-08-15 14:30"

	if out := render(50); !strings.Contains(out, "2.0 KB") || !strings.Contains(out, wantDate) {
		t.Errorf("wide: Render() = %q, want both size and date", out)
	}
	if out := render(30); !strings.Contains(out, "2.0 KB") || strings.Contains(out, wantDate) {
		t.Errorf("medium: Render() = %q, want size but not date", out)
	}
	if out := render(20); strings.Contains(out, "2.0 KB") || strings.Contains(out, wantDate) {
		t.Errorf("narrow: Render() = %q, want neither size nor date", out)
	}
}

func TestFileDelegateShowsNoSizeForDirectories(t *testing.T) {
	dir := fileItem{file: model.FileInfo{Name: "docs", Type: model.FileTypeDir}}
	items := []list.Item{dir}
	delegate := fileDelegate{marked: map[int]bool{}}
	l := list.New(items, delegate, 50, 10)

	var buf bytes.Buffer
	delegate.Render(&buf, l, 0, items[0])
	if out := buf.String(); !strings.Contains(out, "-") {
		t.Errorf("Render() for a directory = %q, want a placeholder instead of a size", out)
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
