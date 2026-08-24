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
	marked := map[string]bool{"report.txt": true}
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

// #52: the acceptance criteria requires this be distinguishable without
// relying on color alone, so the glyph itself (not just the color) has to
// show up in the render.
func TestFileDelegateMarksEntriesUniqueToOneSide(t *testing.T) {
	items := []list.Item{
		fileItem{file: model.FileInfo{Name: "only-here.txt"}},
		fileItem{file: model.FileInfo{Name: "on-both.txt"}},
	}
	delegate := fileDelegate{
		marked:     map[string]bool{},
		uniqueOnly: map[string]bool{"only-here.txt": true},
	}
	l := list.New(items, delegate, 80, 10)

	render := func(i int) string {
		var buf bytes.Buffer
		delegate.Render(&buf, l, i, items[i])
		return buf.String()
	}

	if out := render(0); !strings.Contains(out, iconUnique()) {
		t.Errorf("unique entry: Render() = %q, want the unique indicator", out)
	}
	if out := render(1); strings.Contains(out, iconUnique()) {
		t.Errorf("shared entry: Render() = %q, want no unique indicator", out)
	}
}

// nil (the --highlight-diff flag is off) must drop the indicator column
// rather than reserve a permanently blank one nobody asked for.
func TestFileDelegateOmitsUniqueColumnWhenFlagIsOff(t *testing.T) {
	items := []list.Item{fileItem{file: model.FileInfo{Name: "report.txt"}}}
	delegate := fileDelegate{marked: map[string]bool{}, uniqueOnly: nil}
	l := list.New(items, delegate, 80, 10)

	var buf bytes.Buffer
	delegate.Render(&buf, l, 0, items[0])
	if out := buf.String(); strings.Contains(out, iconUnique()) {
		t.Errorf("Render() with uniqueOnly=nil = %q, want no unique indicator ever", out)
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

	p := NewPanel("Local", true).WithFiles(files, "/tmp")
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
	p := NewPanel("Local", true).WithFiles(files, "/tmp")

	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !p.marked["a.txt"] {
		t.Fatal("space did not mark the file under the cursor")
	}

	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if p.marked["a.txt"] {
		t.Fatal("space did not unmark an already-marked file")
	}
}

// Marks are keyed by filename, not list position: #30 (sort) and #31
// (filter) both reorder or subset what a given index points at, and a
// position-keyed mark would silently follow the wrong file.
func TestMarkFollowsTheFileNotItsPosition(t *testing.T) {
	files := []model.FileInfo{{Name: "a.txt"}, {Name: "b.txt"}, {Name: "c.txt"}}
	p := NewPanel("Local", true).WithFiles(files, "/tmp")

	p.list.Select(1) // b.txt
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})

	// Simulates what a future sort does: reorder p.files and the list's
	// items without going through WithFiles (which would reset marks).
	p.files = []model.FileInfo{{Name: "c.txt"}, {Name: "b.txt"}, {Name: "a.txt"}}

	marked := p.markedFiles()
	if len(marked) != 1 || marked[0].Name != "b.txt" {
		t.Fatalf("markedFiles() after reordering = %v, want just b.txt", marked)
	}
}

// #52: comparison is by name only, and directories and files are treated
// the same way -- a directory unique to one side is just as much "unique"
// as a file is.
func TestUniqueNames(t *testing.T) {
	local := []model.FileInfo{
		{Name: "shared.txt"},
		{Name: "local-only.txt"},
		{Name: "assets", Type: model.FileTypeDir},
	}
	remote := []model.FileInfo{
		{Name: "shared.txt"},
		{Name: "remote-only.txt"},
	}

	localOnly := uniqueNames(local, remote)
	want := map[string]bool{"local-only.txt": true, "assets": true}
	if len(localOnly) != len(want) {
		t.Fatalf("uniqueNames(local, remote) = %v, want %v", localOnly, want)
	}
	for name := range want {
		if !localOnly[name] {
			t.Errorf("uniqueNames(local, remote) missing %q", name)
		}
	}

	remoteOnly := uniqueNames(remote, local)
	if len(remoteOnly) != 1 || !remoteOnly["remote-only.txt"] {
		t.Errorf("uniqueNames(remote, local) = %v, want just remote-only.txt", remoteOnly)
	}
}

func TestSortKeyCyclesColumnAndResetsDirection(t *testing.T) {
	p := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/tmp")
	p.sortDesc = true // s should reset this even mid-cycle

	p, _ = p.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if p.sortBy != sortBySize || p.sortDesc {
		t.Errorf("after s: sortBy=%v sortDesc=%v, want Size ascending", p.sortBy, p.sortDesc)
	}

	p, _ = p.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if p.sortBy != sortByDate {
		t.Errorf("after ss: sortBy=%v, want Date", p.sortBy)
	}

	p, _ = p.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if p.sortBy != sortByName {
		t.Errorf("after sss: sortBy=%v, want Name (wrapped)", p.sortBy)
	}
}

func TestSortFlipReversesWithoutChangingColumn(t *testing.T) {
	p := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/tmp")
	p.sortBy = sortBySize

	p, _ = p.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	if p.sortBy != sortBySize || !p.sortDesc {
		t.Errorf("after S: sortBy=%v sortDesc=%v, want Size descending", p.sortBy, p.sortDesc)
	}

	p, _ = p.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	if p.sortBy != sortBySize || p.sortDesc {
		t.Errorf("after SS: sortBy=%v sortDesc=%v, want Size ascending again", p.sortBy, p.sortDesc)
	}
}

// Re-sorting is triggered by the user looking at a specific file -- jumping
// the cursor back to the top on every keystroke would lose their place.
func TestSortKeepsCursorOnTheSameFile(t *testing.T) {
	files := []model.FileInfo{
		{Name: "b.txt", Size: 200},
		{Name: "a.txt", Size: 300},
		{Name: "c.txt", Size: 100},
	}
	p := NewPanel("Local", true).WithFiles(files, "/tmp")

	p.list.Select(1)                                       // b.txt, in the initial name-sorted order
	p, _ = p.Update(tea.KeyPressMsg{Code: 's', Text: "s"}) // switch to size ascending

	item, ok := p.list.SelectedItem().(fileItem)
	if !ok || item.file.Name != "b.txt" {
		t.Errorf("selected item after resort = %+v, want cursor to stay on b.txt", item)
	}
}

func TestRefreshReloadsTheCurrentPath(t *testing.T) {
	p := NewPanel("Remote", false).WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/srv")

	_, cmd := p.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd == nil {
		t.Fatal("r did not return a command")
	}
	msg, ok := cmd().(NavigateMsg)
	if !ok {
		t.Fatalf("r returned %T, want NavigateMsg", cmd())
	}
	if msg.Panel != "Remote" || msg.Path != "/srv" {
		t.Errorf("r navigated to %+v, want {Remote /srv} (the current path)", msg)
	}
}

func typeInto(p Panel, s string) Panel {
	for _, r := range s {
		p, _ = p.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return p
}

func TestJumpKeyOpensThePathInput(t *testing.T) {
	p := NewPanel("Local", true).WithFiles(nil, "/tmp")

	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	if !p.jumping {
		t.Fatal(": did not open the jump input")
	}
}

func TestJumpEnterNavigatesToAnAbsolutePath(t *testing.T) {
	p := NewPanel("Remote", false).WithFiles(nil, "/srv")
	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	p = typeInto(p, "/var/www")

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.jumping {
		t.Error("jump input still open after Enter")
	}
	if cmd == nil {
		t.Fatal("Enter did not return a command")
	}
	msg, ok := cmd().(NavigateMsg)
	if !ok {
		t.Fatalf("Enter returned %T, want NavigateMsg", cmd())
	}
	if msg.Panel != "Remote" || msg.Path != "/var/www" {
		t.Errorf("navigated to %+v, want {Remote /var/www}", msg)
	}
}

func TestJumpRelativePathResolvesAgainstTheCurrentDir(t *testing.T) {
	p := NewPanel("Remote", false).WithFiles(nil, "/srv/www")
	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	p = typeInto(p, "sub")

	_, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msg := cmd().(NavigateMsg)
	if msg.Path != "/srv/www/sub" {
		t.Errorf("relative jump landed on %q, want /srv/www/sub", msg.Path)
	}
}

// Invalid destinations aren't rejected here: they're not special-cased at
// all. Enter always fires a NavigateMsg, which routes through the same
// loadLocalDir/loadRemoteDir path as ordinary navigation -- a bad path logs
// an error and never sends back a *DirLoadedMsg, so the panel is left where
// it was for free, without this code needing to know what "invalid" means.

func TestJumpEscCancelsWithoutNavigating(t *testing.T) {
	p := NewPanel("Local", true).WithFiles(nil, "/tmp")
	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	p = typeInto(p, "/etc")

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if p.jumping {
		t.Error("jump input still open after Esc")
	}
	if cmd != nil {
		t.Error("Esc returned a command, want nil: cancelling must not navigate")
	}
	if p.path != "/tmp" {
		t.Errorf("path changed to %q after Esc, want it unchanged at /tmp", p.path)
	}
}

func TestJumpEmptyInputCancelsWithoutNavigating(t *testing.T) {
	p := NewPanel("Local", true).WithFiles(nil, "/tmp")
	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.jumping {
		t.Error("jump input still open after Enter on empty input")
	}
	if cmd != nil {
		t.Error("Enter on empty input returned a command, want nil")
	}
}

// The jump input must own every keystroke, including letters that are
// otherwise bound: a path containing "t" or "r" must not trigger transfer
// or refresh while it's being typed.
func TestJumpInputSwallowsOtherwiseBoundKeys(t *testing.T) {
	p := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/tmp")
	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})

	p, _ = p.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	p, _ = p.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	if got := p.jumpInput.Value(); got != "tr" {
		t.Errorf("jumpInput.Value() = %q, want \"tr\" (both keys typed, not acted on)", got)
	}
}

func TestFileDelegateColumnsDegradeWithWidth(t *testing.T) {
	modTime := time.Date(2026, 8, 15, 14, 30, 0, 0, time.UTC)
	file := fileItem{file: model.FileInfo{Name: "report.txt", Size: 2048, ModTime: modTime}}
	items := []list.Item{file}
	delegate := fileDelegate{marked: map[string]bool{}}

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
	delegate := fileDelegate{marked: map[string]bool{}}
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
