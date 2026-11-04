package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/config"
	"github.com/zalando/go-keyring"
)

func keyMsg(s string) tea.KeyPressMsg {
	if s == " " {
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	}
	r := []rune(s)
	return tea.KeyPressMsg{Code: r[0], Text: s}
}

var (
	tab      = tea.KeyPressMsg{Code: tea.KeyTab}
	shiftTab = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	left     = tea.KeyPressMsg{Code: tea.KeyLeft}
	right    = tea.KeyPressMsg{Code: tea.KeyRight}
)

// Tab order must match the visual layout (Protocol, Host, Port, User, Pass):
// it used to follow the field enum's declaration order instead, so tabbing
// from Host skipped over the visually-adjacent Port field.
func TestTabFollowsVisualOrder(t *testing.T) {
	bar := NewConnectionBar()
	want := []connField{fieldHost, fieldPort, fieldUser, fieldPass, fieldKey, fieldProtocol}

	bar, _ = bar.Update(tab) // off the protocol field, onto Host
	for i, field := range want {
		if bar.focused != field {
			t.Fatalf("step %d: focused field %d, want %d", i, bar.focused, field)
		}
		bar, _ = bar.Update(tab)
	}
}

// The protocol field has no text input behind it, so tabbing onto it used to
// focus an uninitialised one and panic.
func TestTabbingReachesEveryFieldAndWrapsAround(t *testing.T) {
	bar := NewConnectionBar()
	start := bar.focused

	for i := 0; i < int(fieldCount); i++ {
		bar, _ = bar.Update(tab)
	}
	if bar.focused != start {
		t.Errorf("a full cycle of tab ended on field %d, want %d", bar.focused, start)
	}

	for i := 0; i < int(fieldCount); i++ {
		bar, _ = bar.Update(shiftTab)
	}
	if bar.focused != start {
		t.Errorf("a full cycle of shift+tab ended on field %d, want %d", bar.focused, start)
	}
}

func TestProtocolCyclesAndPortPlaceholderFollows(t *testing.T) {
	bar := NewConnectionBar()
	if bar.focused != fieldProtocol {
		t.Fatalf("the bar opens on field %d, want the protocol field", bar.focused)
	}

	if got := bar.inputs[fieldPort].Placeholder; got != "21" {
		t.Errorf("port placeholder is %q on FTP, want \"21\"", got)
	}

	bar, _ = bar.Update(right)
	bar, _ = bar.Update(right)
	if bar.protocol != client.SFTP {
		t.Fatalf("two presses of right landed on %s, want SFTP", bar.protocol)
	}
	if got := bar.inputs[fieldPort].Placeholder; got != "22" {
		t.Errorf("port placeholder is %q on SFTP, want \"22\"", got)
	}

	bar, _ = bar.Update(left)
	if bar.protocol != client.FTPS {
		t.Errorf("left from SFTP landed on %s, want FTPS", bar.protocol)
	}
}

// Key.String() prints the space bar as "space", not literal " " -- a case
// that matched on " " silently stopped working when this migrated to v2.
func TestSpaceAlsoCyclesProtocol(t *testing.T) {
	bar := NewConnectionBar()
	start := bar.protocol

	bar, _ = bar.Update(keyMsg(" "))
	if bar.protocol == start {
		t.Errorf("space did not cycle the protocol from %s", start)
	}
}

// Cycling the protocol must not leak keystrokes into the text fields, and typing
// must not reach them while the protocol is focused.
func TestProtocolKeysDoNotReachTheInputs(t *testing.T) {
	bar := NewConnectionBar()
	bar, _ = bar.Update(keyMsg("x"))
	bar, _ = bar.Update(right)

	for f, in := range bar.inputs {
		if in.Value() != "" {
			t.Errorf("field %d holds %q, want it untouched", f, in.Value())
		}
	}

	bar, _ = bar.Update(tab)
	bar, _ = bar.Update(keyMsg("h"))
	if got := bar.inputs[fieldHost].Value(); got != "h" {
		t.Errorf("host holds %q after tabbing to it and typing, want \"h\"", got)
	}
}

// Port only accepts digits -- anything else typed must be swallowed, but
// editing keys (backspace here) still need to reach the input.
func TestPortOnlyAcceptsDigits(t *testing.T) {
	bar := NewConnectionBar()
	bar, _ = bar.Update(tab) // Host
	bar, _ = bar.Update(tab) // Port

	for _, s := range []string{"a", "!", " ", "2", "1"} {
		bar, _ = bar.Update(keyMsg(s))
	}
	if got := bar.inputs[fieldPort].Value(); got != "21" {
		t.Errorf("port holds %q, want \"21\" (letters/symbols/space rejected)", got)
	}

	backspace := tea.KeyPressMsg{Code: tea.KeyBackspace}
	bar, _ = bar.Update(backspace)
	if got := bar.inputs[fieldPort].Value(); got != "2" {
		t.Errorf("backspace should still edit the port field, got %q", got)
	}
}

// A password belongs to the server it was typed or loaded for. Edit the form
// toward another server and the old one must not travel with it.
func focusedOn(bar ConnectionBar, f connField) ConnectionBar {
	bar = bar.blur()
	bar.focused = f
	return bar.focus()
}

func TestEditingTheHostClearsThePassword(t *testing.T) {
	bar := NewConnectionBar()
	bar = fillField(bar, fieldHost, "nas.lan")
	bar = fillField(bar, fieldUser, "ana")
	bar = fillField(bar, fieldPass, "hunter2")

	bar, _ = focusedOn(bar, fieldHost).Update(keyMsg("x"))

	if got := bar.inputs[fieldPass].Value(); got != "" {
		t.Errorf("the password survived a change of host: %q", got)
	}
}

func TestEditingTheUserClearsThePassword(t *testing.T) {
	bar := NewConnectionBar()
	bar = fillField(bar, fieldHost, "nas.lan")
	bar = fillField(bar, fieldUser, "ana")
	bar = fillField(bar, fieldPass, "hunter2")

	bar, _ = focusedOn(bar, fieldUser).Update(keyMsg("x"))

	if got := bar.inputs[fieldPass].Value(); got != "" {
		t.Errorf("the password survived a change of user: %q", got)
	}
}

func TestChangingTheProtocolClearsThePassword(t *testing.T) {
	bar := NewConnectionBar()
	bar = fillField(bar, fieldHost, "nas.lan")
	bar = fillField(bar, fieldPass, "hunter2")

	bar, _ = focusedOn(bar, fieldProtocol).Update(right)

	if got := bar.inputs[fieldPass].Value(); got != "" {
		t.Errorf("the password survived a change of protocol: %q", got)
	}
}

func TestAPasswordLoadedFromTheKeyringDoesNotOutliveItsServer(t *testing.T) {
	keyring.MockInit()
	config.SetSecret(fav, "hunter2")
	bar := NewConnectionBar().fill(fav)
	if got := bar.inputs[fieldPass].Value(); got != "hunter2" {
		t.Fatalf("filling the favorite loaded %q, want its password", got)
	}

	bar, _ = focusedOn(bar, fieldHost).Update(keyMsg("x"))

	if got := bar.inputs[fieldPass].Value(); got != "" {
		t.Errorf("the keyring password stayed in the form after the host changed: %q", got)
	}
}

func TestEditingOnlyThePortOrThePasswordKeepsThePassword(t *testing.T) {
	bar := NewConnectionBar()
	bar = fillField(bar, fieldHost, "nas.lan")
	bar = fillField(bar, fieldUser, "ana")
	bar = fillField(bar, fieldPass, "hunter2")

	bar, _ = focusedOn(bar, fieldPort).Update(keyMsg("2"))
	bar, _ = focusedOn(bar, fieldPass).Update(keyMsg("3"))

	if got := bar.inputs[fieldPass].Value(); got != "hunter23" {
		t.Errorf("the password is %q, want it kept and extended to hunter23", got)
	}
}

func TestTheDialogWarnsOnlyWhenTheProtocolIsPlainFTP(t *testing.T) {
	for _, tc := range []struct {
		protocol client.Protocol
		warns    bool
	}{{client.FTP, true}, {client.FTPS, false}, {client.SFTP, false}} {
		bar := NewConnectionBar()
		bar.protocol = tc.protocol

		view := bar.View(minWidth)

		if got := strings.Contains(view, "Plain FTP is not encrypted"); got != tc.warns {
			t.Errorf("%s: warning shown = %v, want %v", tc.protocol, got, tc.warns)
		}
		if h := lipgloss.Height(view); h > minHeight-2 {
			t.Errorf("%s: the dialog is %d rows, the canvas has %d", tc.protocol, h, minHeight-2)
		}
	}
}
