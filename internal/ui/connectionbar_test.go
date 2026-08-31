package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/client"
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
	want := []connField{fieldHost, fieldPort, fieldUser, fieldPass, fieldProtocol}

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
