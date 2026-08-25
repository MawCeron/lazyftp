package ui

import (
	"errors"
	"fmt"
	"image/color"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
)

// stubClient stands in for a server. Only Disconnect is observed: abandoning an
// attempt has to close whatever the attempt opened.
type stubClient struct {
	disconnected bool
}

func (s *stubClient) Connect(host, user, pass string, port int) error { return nil }
func (s *stubClient) Disconnect() error                               { s.disconnected = true; return nil }
func (s *stubClient) List(path string) ([]model.FileInfo, error)      { return nil, nil }
func (s *stubClient) Mkdir(path string) error                         { return nil }
func (s *stubClient) Upload(local, remote string, p func(int64)) error {
	return nil
}
func (s *stubClient) Download(remote, local string, p func(int64)) error {
	return nil
}

var _ client.Client = (*stubClient)(nil)

func connecting() App {
	a := NewApp(nil, false, nil, "dev", false)
	a.connecting = true
	a.connectSeq = 1
	return a
}

// Unlike "q", Ctrl+C must quit from every focus state -- including the
// connection bar, where "q" is deliberately left to reach the text field.
func TestCtrlCQuitsFromEveryFocusState(t *testing.T) {
	for _, focus := range []focus{focusConnectionBar, focusLocal, focusRemote} {
		a := NewApp(nil, false, nil, "dev", false)
		a.focus = focus
		a.connected = true
		stub := &stubClient{}
		a.client = stub

		// Text is empty, as it is for a real ctrl+c: modifier combos don't
		// produce printable text, and a non-empty Text would short-circuit
		// String() into ignoring Mod entirely.
		_, cmd := a.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		if cmd == nil {
			t.Errorf("focus %d: ctrl+c returned no command", focus)
			continue
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("focus %d: ctrl+c did not quit", focus)
		}
		if !stub.disconnected {
			t.Errorf("focus %d: ctrl+c did not disconnect the client", focus)
		}
	}
}

func TestEscAbandonsAnAttemptInProgress(t *testing.T) {
	a := connecting()

	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	a = model.(App)

	if a.connecting {
		t.Error("the attempt is still marked as running after Esc")
	}
	if a.connectSeq == 1 {
		t.Error("the attempt was not invalidated, so its result would still be taken")
	}
}

// The attempt keeps running after Esc, because neither client library accepts a
// context. Its result must not connect the application behind the user's back.
func TestAnAbandonedAttemptDoesNotConnect(t *testing.T) {
	a := connecting()

	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	a = model.(App)

	stub := &stubClient{}
	model, _ = a.Update(connectedMsg{seq: 1, client: stub, addr: "example.org:21"})
	a = model.(App)

	if a.connected {
		t.Error("an abandoned attempt connected the application")
	}
	if a.client != nil {
		t.Error("an abandoned attempt installed its client")
	}
	if !stub.disconnected {
		t.Error("the abandoned attempt left its connection open")
	}
}

func TestAnAbandonedAttemptDoesNotReportItsError(t *testing.T) {
	a := connecting()

	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	a = model.(App)
	before := len(a.log.entries)

	model, _ = a.Update(connectFailedMsg{seq: 1, err: errors.New("i/o timeout")})
	a = model.(App)

	if len(a.log.entries) != before {
		t.Errorf("an abandoned attempt logged its failure: %q", a.log.entries[before].Message)
	}
}

func TestACurrentAttemptStillConnects(t *testing.T) {
	a := connecting()

	stub := &stubClient{}
	model, _ := a.Update(connectedMsg{seq: 1, client: stub, addr: "example.org:21"})
	a = model.(App)

	if !a.connected {
		t.Fatal("a current attempt did not connect")
	}
	if a.connecting {
		t.Error("the application is still marked as connecting")
	}
	if stub.disconnected {
		t.Error("a current attempt was disconnected")
	}
}

// U/D transfer whichever side has marked files, regardless of focus. With
// nothing marked there's no file to infer, so they must not silently no-op.
func TestDirectTransferKeysRequireMarkedFiles(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.focus = focusLocal

	model, _ := a.Update(tea.KeyPressMsg{Code: 'U', Text: "U"})
	a = model.(App)
	if n := len(a.log.entries); n == 0 || a.log.entries[n-1].Level != LogError {
		t.Fatalf("U with nothing marked in LOCAL did not log an error")
	}

	model, _ = a.Update(tea.KeyPressMsg{Code: 'D', Text: "D"})
	a = model.(App)
	if n := len(a.log.entries); n == 0 || a.log.entries[n-1].Level != LogError {
		t.Fatalf("D with nothing marked in REMOTE did not log an error")
	}
}

// The jump-to-path input is modal like the connection dialog: global
// bindings must not steal a keystroke a typed path would otherwise contain.
// "q" is the sharpest case -- it's also quit.
func TestGlobalKeysDoNotReachThroughAnOpenJumpInput(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.focus = focusLocal

	model, _ := a.Update(tea.KeyPressMsg{Code: ':', Text: ":"})
	a = model.(App)
	if !a.local.jumping {
		t.Fatal(": did not open the jump input")
	}

	model, cmd := a.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	a = model.(App)
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Fatal("q quit the app while the jump input was open")
		}
	}
	if got := a.local.jumpInput.Value(); got != "q" {
		t.Errorf("jumpInput.Value() = %q, want \"q\" (typed, not treated as quit)", got)
	}

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	a = model.(App)
	if a.local.jumping {
		t.Error("jump input still open after Esc")
	}
}

func TestStatusLineReflectsConnectionState(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width = 80

	if got := a.statusLine(); !strings.Contains(got, "OFFLINE") {
		t.Errorf("idle statusLine() = %q, want it to mention not being connected", got)
	}

	a.connecting = true
	a.connectStart = time.Now()
	if got := a.statusLine(); !strings.Contains(got, "CONNECTING") {
		t.Errorf("connecting statusLine() = %q, want it to mention connecting", got)
	}

	a.connecting = false
	a.connected = true
	a.connUser = "admin"
	a.connAddr = "ftp.example.com:21"
	a.connProtocol = client.FTP
	got := a.statusLine()
	for _, want := range []string{"admin", "ftp.example.com:21", "FTP"} {
		if !strings.Contains(got, want) {
			t.Errorf("connected statusLine() = %q, want it to contain %q", got, want)
		}
	}
}

func TestStatusLineShowsMarkedCountOnlyWhenSomethingIsMarked(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width = 80

	if got := a.statusLine(); strings.Contains(got, "MARKED") {
		t.Errorf("nothing marked: statusLine() = %q, want no marked-count pill", got)
	}

	a.local, _ = a.local.WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/tmp")
	a.local, _ = a.local.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})

	if got := a.statusLine(); !strings.Contains(got, "1 MARKED") {
		t.Errorf("one file marked: statusLine() = %q, want a 1 MARKED pill", got)
	}
}

func TestCenterPadding(t *testing.T) {
	cases := []struct {
		width, content int
		wantL, wantR   int
	}{
		{width: 20, content: 10, wantL: 5, wantR: 5},
		{width: 21, content: 10, wantL: 5, wantR: 6}, // odd slack leans right
		{width: 5, content: 10, wantL: 0, wantR: 0},  // doesn't fit: no negative padding
	}
	for _, c := range cases {
		l, r := centerPadding(c.width, c.content)
		if l != c.wantL || r != c.wantR {
			t.Errorf("centerPadding(%d, %d) = (%d, %d), want (%d, %d)", c.width, c.content, l, r, c.wantL, c.wantR)
		}
	}
}

func TestFooterAnchorsAppIdentityRegardlessOfFocus(t *testing.T) {
	a := NewApp(nil, false, nil, "1.2.3", false)
	a.width = 100

	firstHint := map[focus]string{
		focusLocal:         "l/enter",
		focusRemote:        "l/enter",
		focusConnectionBar: "tab",
	}

	for _, focus := range []focus{focusLocal, focusRemote, focusConnectionBar} {
		a.focus = focus
		got := a.hintsView()
		if !strings.Contains(got, "lazyftp") || !strings.Contains(got, "1.2.3") {
			t.Errorf("focus %d: hintsView() = %q, want both identity pills", focus, got)
		}
		if strings.Index(got, "lazyftp") > strings.Index(got, "1.2.3") {
			t.Errorf("focus %d: hintsView() = %q, want lazyftp before the version", focus, got)
		}
		if strings.Index(got, "1.2.3") > strings.Index(got, firstHint[focus]) {
			t.Errorf("focus %d: hintsView() = %q, want the identity pills before the key hints", focus, got)
		}
	}
}

func TestTooSmallViewBelowTheFloor(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)

	a.width, a.height = minWidth-1, minHeight+5
	if got := a.render(); !strings.Contains(got, "too small") {
		t.Errorf("width below the floor: render() = %q, want the too-small message", got)
	}

	a.width, a.height = minWidth+5, minHeight-1
	if got := a.render(); !strings.Contains(got, "too small") {
		t.Errorf("height below the floor: render() = %q, want the too-small message", got)
	}

	a.width, a.height = minWidth, minHeight
	if got := a.render(); strings.Contains(got, "too small") {
		t.Errorf("at exactly the floor: render() = %q, want the normal layout", got)
	}
}

// Below 80 columns, two file panels side by side are too cramped to be
// useful: only the focused one should render, at full width.
func TestNarrowWidthShowsOnlyTheFocusedPanel(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width, a.height = 70, 24
	a.focus = focusLocal

	out := a.render()
	if !strings.Contains(out, "Local") {
		t.Error("narrow width with Local focused: Local panel missing from render()")
	}
	if strings.Contains(out, "Remote") {
		t.Error("narrow width with Local focused: Remote panel should not render")
	}

	a.focus = focusRemote
	out = a.render()
	if !strings.Contains(out, "Remote") {
		t.Error("narrow width with Remote focused: Remote panel missing from render()")
	}
	if strings.Contains(out, "Local") {
		t.Error("narrow width with Remote focused: Local panel should not render")
	}
}

func TestStandardWidthShowsBothPanels(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width, a.height = 80, 24
	a.focus = focusLocal // panels render blank behind the connection overlay

	out := a.render()
	if !strings.Contains(out, "Local") || !strings.Contains(out, "Remote") {
		t.Errorf("80 columns should show both panels side by side; render() = %q", out)
	}
}

// #52 is opt-in: with --highlight-diff off (NewApp's default), a name
// unique to one side must not render any differently than it always has.
func TestHighlightDiffOnlyActivatesWithTheFlag(t *testing.T) {
	newApp := func(highlightDiff bool) App {
		a := NewApp(nil, false, nil, "dev", highlightDiff)
		a.width, a.height = 100, 30
		a.focus = focusLocal
		a.local, _ = a.local.WithFiles([]model.FileInfo{{Name: "only-local.txt"}}, "/a")
		a.remote, _ = a.remote.WithFiles([]model.FileInfo{{Name: "only-remote.txt"}}, "/b")
		return a
	}

	off := newApp(false)
	if out := off.render(); strings.Contains(out, iconUnique()) {
		t.Errorf("--highlight-diff off: render() = %q, want no unique indicator", out)
	}

	on := newApp(true)
	if out := on.render(); !strings.Contains(out, iconUnique()) {
		t.Errorf("--highlight-diff on: render() = %q, want a unique indicator", out)
	}
}

// #52 extension: same name on both sides, different size, still gated by
// the same flag.
func TestHighlightDiffMarksSizeMismatch(t *testing.T) {
	newApp := func(highlightDiff bool) App {
		a := NewApp(nil, false, nil, "dev", highlightDiff)
		a.width, a.height = 100, 30
		a.focus = focusLocal
		a.local, _ = a.local.WithFiles([]model.FileInfo{{Name: "shared.txt", Size: 100}}, "/a")
		a.remote, _ = a.remote.WithFiles([]model.FileInfo{{Name: "shared.txt", Size: 200}}, "/b")
		return a
	}

	off := newApp(false)
	if out := off.render(); strings.Contains(out, iconSizeDiffers()) {
		t.Errorf("--highlight-diff off: render() = %q, want no size-differs indicator", out)
	}

	on := newApp(true)
	if out := on.render(); !strings.Contains(out, iconSizeDiffers()) {
		t.Errorf("--highlight-diff on: render() = %q, want a size-differs indicator", out)
	}
}

// Tab alone used to be its own reverse when there were only two panels
// total. Once Log (and now Processes) needed a place in the cycle too, a
// single Tab-only cycle stopped being reversible with itself -- so Tab
// stays within whichever group has focus, and Shift+Tab is what moves
// between the Local/Remote group and the Log/Processes group.
func TestTabStaysWithinItsGroup(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.focus = focusLocal

	want := []focus{focusRemote, focusLocal}
	for _, w := range want {
		model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		a = model.(App)
		if a.focus != w {
			t.Fatalf("after Tab, focus = %v, want %v", a.focus, w)
		}
	}

	a.focus = focusLog
	want = []focus{focusProcesses, focusLog}
	for _, w := range want {
		model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		a = model.(App)
		if a.focus != w {
			t.Fatalf("after Tab, focus = %v, want %v", a.focus, w)
		}
	}
}

func TestShiftTabSwitchesGroups(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.focus = focusLocal

	model, _ := a.Update(shiftTab)
	a = model.(App)
	if a.focus != focusProcesses {
		t.Fatalf("Shift+Tab from Local: focus = %v, want focusProcesses", a.focus)
	}

	a.focus = focusRemote
	model, _ = a.Update(shiftTab)
	a = model.(App)
	if a.focus != focusProcesses {
		t.Fatalf("Shift+Tab from Remote: focus = %v, want focusProcesses", a.focus)
	}

	a.focus = focusLog
	model, _ = a.Update(shiftTab)
	a = model.(App)
	if a.focus != focusLocal {
		t.Fatalf("Shift+Tab from Log: focus = %v, want focusLocal", a.focus)
	}

	a.focus = focusProcesses
	model, _ = a.Update(shiftTab)
	a = model.(App)
	if a.focus != focusLocal {
		t.Fatalf("Shift+Tab from Processes: focus = %v, want focusLocal", a.focus)
	}
}

// Scrolling keys must reach the Processes panel's viewport only while it
// has focus, mirroring the same isolation Log already gets.
func TestProcessesScrollKeysOnlyReachTheViewportWhenFocused(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width, a.height = 100, 30
	a.processes = a.processes.SetSize(20, 6) // small: guarantees scrolling kicks in
	for i := range 20 {
		a.processes = a.processes.AddTransfer(shared.Transfer{
			Filename: fmt.Sprintf("file-%02d.txt", i),
			Total:    100,
		})
	}

	a.focus = focusRemote
	model, _ := a.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	a = model.(App)
	if !a.processes.viewport.AtBottom() {
		t.Fatal("Processes' viewport scrolled from a key press while Remote had focus")
	}

	a.focus = focusProcesses
	model, _ = a.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	a = model.(App)
	if a.processes.viewport.AtBottom() {
		t.Error("k did not scroll the Processes viewport while it had focus")
	}
}

// Scrolling keys must reach the Log panel's viewport only while it has
// focus, the same isolation Local/Remote already get from each other.
func TestLogScrollKeysOnlyReachTheViewportWhenFocused(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width, a.height = 100, 30
	a.log = a.log.SetSize(20, 6) // small: guarantees scrolling kicks in
	for i := range 20 {
		a.log = a.log.Add(fmt.Sprintf("entry %d", i), LogInfo)
	}

	a.focus = focusLocal
	model, _ := a.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	a = model.(App)
	if !a.log.viewport.AtBottom() {
		t.Fatal("Log's viewport scrolled from a key press while Local had focus")
	}

	a.focus = focusLog
	model, _ = a.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	a = model.(App)
	if a.log.viewport.AtBottom() {
		t.Error("k did not scroll the Log viewport while it had focus")
	}
}

// The connection form floats over the panels rather than replacing them: it
// should only appear in the rendered frame while it holds focus.
func TestConnectionOverlayOnlyAppearsWhenFocused(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width, a.height = 80, 24

	a.focus = focusConnectionBar
	if got := a.render(); !strings.Contains(got, "Connection") {
		t.Error("render() with the connection bar focused does not show the overlay")
	}

	a.focus = focusLocal
	if got := a.render(); strings.Contains(got, "Connection") {
		t.Error("render() with local focused still shows the connection overlay")
	}
}

// The connection dialog is a fixed-size form, not a mode the panels stay
// visible under: showing them was cutting their text off mid-word wherever
// the floating dialog happened to overlap a panel.
func TestConnectionOverlayBlanksThePanels(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width, a.height = 80, 24
	a.focus = focusConnectionBar

	out := a.render()
	if strings.Contains(out, "Local") || strings.Contains(out, "Remote") {
		t.Errorf("connection overlay should blank the panels behind it; render() = %q", out)
	}
}

// tea.RequestBackgroundColor has no timeout of its own -- a terminal that
// never answers the OSC 11 query would otherwise leave the theme stuck
// unresolved forever. themeFallbackMsg is App's own timeout for that case.
func TestThemeFallbackAssumesDarkWhenTerminalNeverResponds(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	if a.themeResolved {
		t.Fatal("themeResolved should start false")
	}

	model, _ := a.Update(themeFallbackMsg{})
	a = model.(App)

	if !a.themeResolved {
		t.Error("themeFallbackMsg should mark the theme resolved")
	}
}

// A real reply can still arrive after the fallback already fired; it should
// still take effect rather than being ignored as "already resolved".
func TestABackgroundColorMsgStillAppliesAfterTheFallbackFired(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)

	model, _ := a.Update(themeFallbackMsg{})
	a = model.(App)
	SetTheme(true)
	darkPrimary := colorPrimary

	model, _ = a.Update(tea.BackgroundColorMsg{Color: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}})
	_ = model.(App)

	if colorPrimary == darkPrimary {
		t.Error("a late BackgroundColorMsg should still override the fallback's dark guess")
	}
}

// Once the real background is known, a fallback tick that arrives after it
// (Init batches both commands, so ordering isn't guaranteed) must not stomp
// back over it.
func TestThemeFallbackYieldsToAnAlreadyResolvedTheme(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)

	model, _ := a.Update(tea.BackgroundColorMsg{Color: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}})
	a = model.(App)
	lightPrimary := colorPrimary

	model, _ = a.Update(themeFallbackMsg{})
	_ = model.(App)

	if colorPrimary != lightPrimary {
		t.Error("the fallback should not override an already-resolved theme")
	}
}

// Rendering reaches lipgloss and the bubbles list through several derived
// widths and heights, any one of which can go negative before the others do.
// Sweeping the sizes covers the arithmetic without having to find each one.
func TestRenderingSurvivesAnyTerminalSize(t *testing.T) {
	files := []model.FileInfo{
		{Name: "a-rather-long-file-name-to-truncate.txt"},
		{Name: "dir", Type: model.FileTypeDir},
	}

	for w := 0; w <= 24; w++ {
		for h := 0; h <= 24; h++ {
			t.Run(fmt.Sprintf("%dx%d", w, h), func(t *testing.T) {
				a := NewApp(nil, false, nil, "dev", false)
				a.local, _ = a.local.WithFiles(files, "/a/deep/enough/path/to/be/truncated")
				a.remote, _ = a.remote.WithFiles(files, "/another/path")

				model, _ := a.Update(tea.WindowSizeMsg{Width: w, Height: h})
				_ = model.(App).View()
			})
		}
	}
}

// The bug this guards against: every other global key (quit, help, upload,
// download) checks whether the focused panel is capturing filter input
// before acting, but Tab -- which also switches focus -- didn't. Since Tab
// is a natural key to reach for out of habit while typing, pressing it
// mid-filter jumped to the other panel and left the first one's filter
// half-finished, unresponsive to open/mark/transfer/refresh until someone
// tabbed back and explicitly finished or cancelled the filter.
func TestTabDoesNotSwitchFocusWhileFiltering(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width, a.height = 80, 24
	a.focus = focusLocal
	a.local, _ = a.local.WithFiles([]model.FileInfo{{Name: "apple.txt"}, {Name: "banana.txt"}}, "/tmp")

	model, _ := a.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	a = model.(App)
	if !a.local.Filtering() {
		t.Fatal("/ did not start filtering on the focused panel")
	}

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	a = model.(App)

	if a.focus != focusLocal {
		t.Error("tab switched focus away from a panel while it was mid-filter")
	}
	if !a.local.Filtering() {
		t.Error("tab should have reached the list (which owns tab while filtering), not been swallowed")
	}
}
