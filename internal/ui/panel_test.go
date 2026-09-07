package ui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/model"
)

// runFilterCmd runs a command returned from a Panel and feeds any resulting
// message back into Update, mirroring what the real Bubble Tea runtime does
// automatically. Tests that drive the list's filter need this: filtering
// (SetItems included, see #31) relies on the framework running the commands
// it returns, which doesn't happen on its own inside a unit test.
func runFilterCmd(p Panel, cmd tea.Cmd) Panel {
	if cmd == nil {
		return p
	}
	msg := cmd()
	if msg == nil {
		return p
	}
	p, _ = p.Update(msg)
	return p
}

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

// #52 extension: same distinguishable-without-color-alone requirement, its
// own glyph so it doesn't get read as "missing" (iconUnique's meaning).
func TestFileDelegateMarksSizeDiffersDistinctFromUnique(t *testing.T) {
	items := []list.Item{
		fileItem{file: model.FileInfo{Name: "changed.txt"}},
		fileItem{file: model.FileInfo{Name: "same.txt"}},
	}
	delegate := fileDelegate{
		marked:      map[string]bool{},
		uniqueOnly:  map[string]bool{},
		sizeDiffers: map[string]bool{"changed.txt": true},
	}
	l := list.New(items, delegate, 80, 10)

	render := func(i int) string {
		var buf bytes.Buffer
		delegate.Render(&buf, l, i, items[i])
		return buf.String()
	}

	out := render(0)
	if !strings.Contains(out, iconSizeDiffers()) {
		t.Errorf("changed entry: Render() = %q, want the size-differs indicator", out)
	}
	if strings.Contains(out, iconUnique()) {
		t.Errorf("changed entry: Render() = %q, want the unique indicator NOT to also show", out)
	}

	if out := render(1); strings.Contains(out, iconSizeDiffers()) {
		t.Errorf("unchanged entry: Render() = %q, want no size-differs indicator", out)
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

	p, _ := NewPanel("Local", true).WithFiles(files, "/tmp")
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
	p, _ := NewPanel("Local", true).WithFiles(files, "/tmp")

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
	p, _ := NewPanel("Local", true).WithFiles(files, "/tmp")

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

// #52 extension: size is an exact byte count from both sides, unlike a
// timestamp, so comparing it doesn't carry the reliability problem that
// kept the original feature to presence-by-name only.
func TestSizeDiffers(t *testing.T) {
	local := []model.FileInfo{
		{Name: "same.txt", Size: 100},
		{Name: "changed.txt", Size: 200},
		{Name: "local-only.txt", Size: 50},
		{Name: "assets", Type: model.FileTypeDir, Size: 999}, // dirs excluded regardless of Size
	}
	remote := []model.FileInfo{
		{Name: "same.txt", Size: 100},
		{Name: "changed.txt", Size: 250},
		{Name: "assets", Type: model.FileTypeDir, Size: 1},
	}

	got := sizeDiffers(local, remote)
	if len(got) != 1 || !got["changed.txt"] {
		t.Errorf("sizeDiffers(local, remote) = %v, want just changed.txt", got)
	}
}

func TestSortKeyCyclesColumnAndResetsDirection(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/tmp")
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
	p, _ := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/tmp")
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
	p, _ := NewPanel("Local", true).WithFiles(files, "/tmp")

	p.list.Select(1)                                       // b.txt, in the initial name-sorted order
	p, _ = p.Update(tea.KeyPressMsg{Code: 's', Text: "s"}) // switch to size ascending

	item, ok := p.list.SelectedItem().(fileItem)
	if !ok || item.file.Name != "b.txt" {
		t.Errorf("selected item after resort = %+v, want cursor to stay on b.txt", item)
	}
}

func TestRefreshReloadsTheCurrentPath(t *testing.T) {
	p, _ := NewPanel("Remote", false).WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/srv")

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
	p, _ := NewPanel("Local", true).WithFiles(nil, "/tmp")

	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	if !p.jumping {
		t.Fatal(": did not open the jump input")
	}
}

func TestJumpEnterNavigatesToAnAbsolutePath(t *testing.T) {
	p, _ := NewPanel("Remote", false).WithFiles(nil, "/srv")
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
	p, _ := NewPanel("Remote", false).WithFiles(nil, "/srv/www")
	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	p = typeInto(p, "sub")

	_, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msg := cmd().(NavigateMsg)
	if msg.Path != "/srv/www/sub" {
		t.Errorf("relative jump landed on %q, want /srv/www/sub", msg.Path)
	}
}

func TestJumpTildeExpandsToHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available in this environment")
	}

	p, _ := NewPanel("Local", true).WithFiles(nil, "/somewhere/else")
	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	p = typeInto(p, "~")

	_, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msg := cmd().(NavigateMsg)
	if msg.Path != filepath.Clean(home) {
		t.Errorf("bare ~ resolved to %q, want %q", msg.Path, filepath.Clean(home))
	}
}

func TestJumpTildeSlashResolvesRelativeToHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available in this environment")
	}

	p, _ := NewPanel("Local", true).WithFiles(nil, "/somewhere/else")
	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	p = typeInto(p, "~/Documents")

	_, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msg := cmd().(NavigateMsg)
	want := filepath.Join(home, "Documents")
	if msg.Path != want {
		t.Errorf("~/Documents resolved to %q, want %q", msg.Path, want)
	}
}

// A Windows user types "~\Documents", not "~/Documents".
func TestJumpTildeBackslashResolvesRelativeToHomeOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows path semantics")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available in this environment")
	}

	p, _ := NewPanel("Local", true).WithFiles(nil, `C:\somewhere\else`)
	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	p = typeInto(p, `~\Documents`)

	_, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msg := cmd().(NavigateMsg)
	want := filepath.Join(home, "Documents")
	if msg.Path != want {
		t.Errorf(`~\Documents resolved to %q, want %q`, msg.Path, want)
	}
}

// "~" has no universal meaning over FTP/SFTP, so the remote panel must treat
// it as a literal path segment, not expand it.
func TestJumpTildeIsLiteralOnTheRemotePanel(t *testing.T) {
	p, _ := NewPanel("Remote", false).WithFiles(nil, "/srv")
	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	p = typeInto(p, "~/Documents")

	_, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msg := cmd().(NavigateMsg)
	if msg.Path != "/srv/~/Documents" {
		t.Errorf("remote ~ was expanded to %q, want it treated literally as /srv/~/Documents", msg.Path)
	}
}

// Invalid destinations aren't rejected here: they're not special-cased at
// all. Enter always fires a NavigateMsg, which routes through the same
// loadLocalDir/loadRemoteDir path as ordinary navigation -- a bad path logs
// an error and never sends back a *DirLoadedMsg, so the panel is left where
// it was for free, without this code needing to know what "invalid" means.

// Remote, not Local: local paths follow the host's own separator rules
// (filepath.Clean turns "/tmp" into "\tmp" on Windows), and this test only
// cares that Esc leaves the path untouched, not what that path looks like.
func TestJumpEscCancelsWithoutNavigating(t *testing.T) {
	p, _ := NewPanel("Remote", false).WithFiles(nil, "/tmp")
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
	p, _ := NewPanel("Local", true).WithFiles(nil, "/tmp")
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
	p, _ := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/tmp")
	p, _ = p.Update(tea.KeyPressMsg{Code: ':', Text: ":"})

	p, _ = p.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	p, _ = p.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	if got := p.jumpInput.Value(); got != "tr" {
		t.Errorf("jumpInput.Value() = %q, want \"tr\" (both keys typed, not acted on)", got)
	}
}

func TestMkdirKeyOpensTheNameInput(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles(nil, "/tmp")

	p, _ = p.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	if !p.creatingDir {
		t.Fatal("ctrl+n did not open the new-directory input")
	}
}

func TestMkdirEnterCreatesInsideCurrentDir(t *testing.T) {
	p, _ := NewPanel("Remote", false).WithFiles(nil, "/srv/www")
	p, _ = p.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	p = typeInto(p, "newdir")

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.creatingDir {
		t.Error("new-directory input still open after Enter")
	}
	if cmd == nil {
		t.Fatal("Enter did not return a command")
	}
	msg, ok := cmd().(mkdirMsg)
	if !ok {
		t.Fatalf("Enter returned %T, want mkdirMsg", cmd())
	}
	if msg.Panel != "Remote" || msg.Path != "/srv/www/newdir" {
		t.Errorf("created %+v, want {Remote /srv/www/newdir}", msg)
	}
}

func TestMkdirEscCancelsWithoutCreating(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles(nil, "/tmp")
	p, _ = p.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	p = typeInto(p, "newdir")

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if p.creatingDir {
		t.Error("new-directory input still open after Esc")
	}
	if cmd != nil {
		t.Error("Esc returned a command, want nil: cancelling must not create a directory")
	}
}

func TestMkdirEmptyInputCancelsWithoutCreating(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles(nil, "/tmp")
	p, _ = p.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.creatingDir {
		t.Error("new-directory input still open after Enter on empty input")
	}
	if cmd != nil {
		t.Error("Enter on empty input returned a command, want nil")
	}
}

func TestRenameKeyOpensTheNameInputPrefilled(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "old.txt"}}, "/tmp")

	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	if !p.renaming {
		t.Fatal("F2 did not open the rename input")
	}
	if got := p.renameInput.Value(); got != "old.txt" {
		t.Errorf("renameInput.Value() = %q, want %q (pre-filled with the current name)", got, "old.txt")
	}
}

func TestRenameEnterRenamesToNewName(t *testing.T) {
	p, _ := NewPanel("Remote", false).WithFiles([]model.FileInfo{{Name: "old.txt"}}, "/srv/www")
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	p.renameInput.SetValue("new.txt")

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.renaming {
		t.Error("rename input still open after Enter")
	}
	if cmd == nil {
		t.Fatal("Enter did not return a command")
	}
	msg, ok := cmd().(renameMsg)
	if !ok {
		t.Fatalf("Enter returned %T, want renameMsg", cmd())
	}
	want := renameMsg{Panel: "Remote", OldPath: "/srv/www/old.txt", NewPath: "/srv/www/new.txt"}
	if msg != want {
		t.Errorf("renamed %+v, want %+v", msg, want)
	}
}

func TestRenameEscCancelsWithoutRenaming(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "old.txt"}}, "/tmp")
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	p.renameInput.SetValue("new.txt")

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if p.renaming {
		t.Error("rename input still open after Esc")
	}
	if cmd != nil {
		t.Error("Esc returned a command, want nil: cancelling must not rename")
	}
}

func TestRenameEmptyInputCancelsWithoutRenaming(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "old.txt"}}, "/tmp")
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	p.renameInput.SetValue("")

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.renaming {
		t.Error("rename input still open after Enter on empty input")
	}
	if cmd != nil {
		t.Error("Enter on empty input returned a command, want nil")
	}
}

// Confirming without changing the pre-filled name is a no-op, not a rename
// to itself.
func TestRenameUnchangedNameDoesNothing(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "old.txt"}}, "/tmp")
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyF2})

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.renaming {
		t.Error("rename input still open after confirming an unchanged name")
	}
	if cmd != nil {
		t.Error("confirming an unchanged name returned a command, want nil")
	}
}

func TestRenameKeyWithNoSelectionDoesNothing(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles(nil, "/tmp")

	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	if p.renaming {
		t.Error("F2 opened the rename input with nothing selected")
	}
}

func TestDeleteKeyShowsConfirmationForSelectedFile(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/tmp")

	p, cmd := p.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if !p.deleting {
		t.Fatal("d did not show the delete confirmation")
	}
	if cmd != nil {
		t.Error("d returned a command, want nil: it must only arm the confirmation")
	}
	if len(p.deletingFiles) != 1 || p.deletingFiles[0].Name != "a.txt" {
		t.Errorf("deletingFiles = %+v, want [a.txt]", p.deletingFiles)
	}
}

func TestDeleteEnterEmitsTargetsForMarkedFiles(t *testing.T) {
	files := []model.FileInfo{
		{Name: "a.txt"},
		{Name: "b.txt"},
		{Name: "sub", Type: model.FileTypeDir},
	}
	p, _ := NewPanel("Remote", false).WithFiles(files, "/srv")
	// Mark a.txt and sub, leave b.txt unmarked.
	p.marked["a.txt"] = true
	p.marked["sub"] = true

	p, _ = p.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if len(p.deletingFiles) != 2 {
		t.Fatalf("deletingFiles = %+v, want the 2 marked files, not the cursor file", p.deletingFiles)
	}

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.deleting {
		t.Error("delete confirmation still open after Enter")
	}
	if cmd == nil {
		t.Fatal("Enter did not return a command")
	}
	msg, ok := cmd().(deleteMsg)
	if !ok {
		t.Fatalf("Enter returned %T, want deleteMsg", cmd())
	}
	want := deleteMsg{Panel: "Remote", Targets: []deleteTarget{
		{Path: "/srv/a.txt", IsDir: false},
		{Path: "/srv/sub", IsDir: true},
	}}
	if msg.Panel != want.Panel || len(msg.Targets) != len(want.Targets) {
		t.Fatalf("deleted %+v, want %+v", msg, want)
	}
	for _, wantTarget := range want.Targets {
		found := false
		for _, got := range msg.Targets {
			if got == wantTarget {
				found = true
			}
		}
		if !found {
			t.Errorf("targets %+v, missing %+v", msg.Targets, wantTarget)
		}
	}
}

func TestDeleteEscCancelsWithoutDeleting(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/tmp")
	p, _ = p.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if p.deleting {
		t.Error("delete confirmation still open after Esc")
	}
	if cmd != nil {
		t.Error("Esc returned a command, want nil: cancelling must not delete")
	}
}

func TestDeleteKeyWithNoSelectionDoesNothing(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles(nil, "/tmp")

	p, _ = p.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if p.deleting {
		t.Error("d showed the delete confirmation with nothing to delete")
	}
}

// The confirmation prompt must own every keystroke: a "t" typed by mistake
// (or a genuine attempt to abandon the prompt via some other binding) must
// not fall through to transfer, mark, or any other action.
func TestDeleteConfirmationSwallowsOtherwiseBoundKeys(t *testing.T) {
	p, _ := NewPanel("Local", true).WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/tmp")
	p, _ = p.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})

	p, cmd := p.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	if !p.deleting {
		t.Error("delete confirmation closed by an unrelated key")
	}
	if cmd != nil {
		t.Error("an unrelated key while confirming delete returned a command, want nil")
	}
}

// The bug this guards against: WithFiles called list.SetItems but discarded
// the command it returns. With filtering disabled that was harmless, but
// once filtering was enabled (#31) the list needs that command run to
// rebuild its filtered view against the new items -- otherwise a panel
// reloaded while a filter is active (refresh, navigating into a matched
// directory, or a completed transfer reloading both panels) shows no files
// at all until the filter is cleared and re-typed.
func TestReloadWhileFilteredKeepsMatchingItemsVisible(t *testing.T) {
	files := []model.FileInfo{{Name: "apple.txt"}, {Name: "banana.txt"}, {Name: "cherry.txt"}}
	p, _ := NewPanel("Local", true).WithFiles(files, "/tmp")
	p = p.SetSize(40, 10)

	p, cmd := p.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	p = runFilterCmd(p, cmd)
	if !p.Filtering() {
		t.Fatal("/ did not start filtering")
	}

	for _, k := range []rune{'a', 'p', 'p', 'l', 'e'} {
		p, cmd = p.Update(tea.KeyPressMsg{Code: k, Text: string(k)})
		p = runFilterCmd(p, cmd)
	}

	if len(p.list.VisibleItems()) == 0 {
		t.Fatal("typing a matching filter query left no visible items")
	}

	// Simulate a reload while filtered: same path as refresh, navigating
	// into a matched subdirectory, or a completed transfer.
	reloadFiles := []model.FileInfo{{Name: "apple.txt"}, {Name: "banana.txt"}}
	p, cmd = p.WithFiles(reloadFiles, "/tmp")
	p = runFilterCmd(p, cmd)

	if len(p.list.VisibleItems()) == 0 {
		t.Fatal("reloading while filtered left no visible items -- the WithFiles command was likely dropped")
	}
}

func TestToggleHiddenFilesShowsAndHidesDotfiles(t *testing.T) {
	files := []model.FileInfo{
		{Name: "visible.txt"},
		{Name: ".hidden", IsHidden: true},
	}
	p, _ := NewPanel("Local", true).WithFiles(files, "/tmp")

	if len(p.list.Items()) != 2 {
		t.Fatalf("hidden files should be visible by default, got %d items", len(p.list.Items()))
	}

	p, _ = p.Update(tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl})
	if len(p.list.Items()) != 1 {
		t.Fatalf("ctrl+h should hide dotfiles, got %d items", len(p.list.Items()))
	}
	if p.list.Items()[0].(fileItem).file.Name != "visible.txt" {
		t.Error("the remaining item after hiding dotfiles should be the non-hidden file")
	}

	p, _ = p.Update(tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl})
	if len(p.list.Items()) != 2 {
		t.Fatalf("a second ctrl+h should show dotfiles again, got %d items", len(p.list.Items()))
	}
}

// Toggling hidden files while the cursor sits on a file that remains visible
// must not move the cursor -- only files() shrinking should ever do that.
func TestToggleHiddenKeepsCursorOnTheSameFile(t *testing.T) {
	files := []model.FileInfo{
		{Name: ".hidden", IsHidden: true},
		{Name: "kept.txt"},
	}
	p, _ := NewPanel("Local", true).WithFiles(files, "/tmp")
	p.list.Select(1) // "kept.txt"

	p, _ = p.Update(tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl})
	item, ok := p.list.SelectedItem().(fileItem)
	if !ok || item.file.Name != "kept.txt" {
		t.Errorf("selection after hiding dotfiles = %+v, want kept.txt", item.file)
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

// A marked or selected row renders name/size/date all in the same
// Reverse(true) style, meant to read as one solid highlighted block. The
// separators between them used to be literal, unstyled spaces -- rendered
// outside any style, so they broke the block into visible gaps at each
// column boundary instead of staying part of the reversed span.
func TestHighlightedRowHasNoUnstyledGaps(t *testing.T) {
	modTime := time.Date(2026, 8, 15, 14, 30, 0, 0, time.UTC)
	file := fileItem{file: model.FileInfo{Name: "report.txt", Size: 2048, ModTime: modTime}}
	items := []list.Item{file}

	render := func(marked map[string]bool) string {
		delegate := fileDelegate{marked: marked}
		l := list.New(items, delegate, 50, 10)
		var buf bytes.Buffer
		delegate.Render(&buf, l, 0, items[0])
		return buf.String()
	}

	// An SGR reset immediately followed by a bare space and a fresh escape
	// is exactly the gap: the space sits between two styled spans instead of
	// inside one. Checked from the name onward -- the cursor/mark icon
	// prefix has its own single space before the name that isn't part of
	// this row's highlighted block and was never the bug.
	const gap = "\x1b[m \x1b["

	out := render(map[string]bool{"report.txt": true})
	idx := strings.Index(out, "report.txt")
	if idx < 0 {
		t.Fatalf("rendered output doesn't contain the file name: %q", out)
	}
	if strings.Contains(out[idx:], gap) {
		t.Errorf("marked row has an unstyled gap: %q", out)
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
