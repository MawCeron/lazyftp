package ui

import (
	"testing"

	"github.com/MawCeron/lazyftp/internal/client"
	tea "github.com/charmbracelet/bubbletea"
)

func key(s string) tea.KeyMsg {
	if s == " " {
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

var (
	tab      = tea.KeyMsg{Type: tea.KeyTab}
	shiftTab = tea.KeyMsg{Type: tea.KeyShiftTab}
	left     = tea.KeyMsg{Type: tea.KeyLeft}
	right    = tea.KeyMsg{Type: tea.KeyRight}
)

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

// Cycling the protocol must not leak keystrokes into the text fields, and typing
// must not reach them while the protocol is focused.
func TestProtocolKeysDoNotReachTheInputs(t *testing.T) {
	bar := NewConnectionBar()
	bar, _ = bar.Update(key("x"))
	bar, _ = bar.Update(right)

	for f, in := range bar.inputs {
		if in.Value() != "" {
			t.Errorf("field %d holds %q, want it untouched", f, in.Value())
		}
	}

	bar, _ = bar.Update(tab)
	bar, _ = bar.Update(key("h"))
	if got := bar.inputs[fieldHost].Value(); got != "h" {
		t.Errorf("host holds %q after tabbing to it and typing, want \"h\"", got)
	}
}
