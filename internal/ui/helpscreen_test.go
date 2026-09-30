package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func helpApp(w, h int) App {
	a := NewApp(nil, false, nil, "dev", false)
	a.width, a.height, a.focus = w, h, focusLocal
	return a
}

func press(a App, msg tea.Msg) App {
	got, _ := a.Update(msg)
	return got.(App)
}

func canvasHeight(a App) int {
	_, panelH, bottomH := a.heights()
	return panelH + bottomH
}

// Scrolling has to reach every line: at the floor the reference is taller than
// the screen, which is the whole reason it scrolls.
func TestEveryHelpLineIsReachableAtTheFloor(t *testing.T) {
	a := helpApp(minWidth, minHeight)
	h := canvasHeight(a)
	_, height, lines := helpScreenLayout(a.width, h)
	if len(lines) <= height-2 {
		t.Skip("the reference fits the floor; nothing to scroll")
	}

	seen := map[string]bool{}
	for off := 0; off <= helpScrollMax(a.width, h); off++ {
		for _, l := range strings.Split(helpScreenView(a.width, h, off), "\n") {
			seen[l] = true
		}
	}
	for i, l := range lines {
		found := false
		for s := range seen {
			if strings.Contains(s, strings.TrimRight(l, " ")) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("help line %d %q is never on screen", i, l)
		}
	}
}

func TestHelpScrollKeysMoveAndEscCloses(t *testing.T) {
	a := helpApp(minWidth, minHeight)
	a = press(a, keyMsg("?"))
	if !a.helpOpen || a.helpOffset != 0 {
		t.Fatalf("open=%v offset=%d", a.helpOpen, a.helpOffset)
	}

	a = press(a, keyMsg("j"))
	if a.helpOffset != 1 {
		t.Fatalf("j: offset %d, want 1", a.helpOffset)
	}
	a = press(a, tea.KeyPressMsg{Code: tea.KeyPgDown})
	if a.helpOffset <= 1 {
		t.Fatalf("pgdn: offset %d", a.helpOffset)
	}
	for i := 0; i < 500; i++ {
		a = press(a, tea.KeyPressMsg{Code: tea.KeyPgDown})
	}
	if max := helpScrollMax(a.width, canvasHeight(a)); a.helpOffset != max {
		t.Fatalf("scrolled to %d, the end is %d", a.helpOffset, max)
	}
	a = press(a, keyMsg("k"))
	a = press(a, tea.KeyPressMsg{Code: tea.KeyPgUp})
	if a.helpOffset < 0 {
		t.Fatalf("offset %d", a.helpOffset)
	}
	for i := 0; i < 500; i++ {
		a = press(a, keyMsg("k"))
	}
	if a.helpOffset != 0 {
		t.Fatalf("scrolled past the top: %d", a.helpOffset)
	}

	a = press(a, keyMsg("j"))
	a = press(a, esc)
	if a.helpOpen {
		t.Fatal("esc did not close the help")
	}
	if a = press(a, keyMsg("?")); a.helpOffset != 0 {
		t.Fatalf("reopened at offset %d, want the top", a.helpOffset)
	}
	if a = press(a, keyMsg("?")); a.helpOpen {
		t.Fatal("? did not close the help")
	}
}

func TestHelpIsWrappedToTheBoxNotClipped(t *testing.T) {
	for _, w := range []int{minWidth, 40} {
		a := helpApp(w, minHeight)
		width, _, lines := helpScreenLayout(a.width, canvasHeight(a))
		inner := borderInteriorWidth(width)
		for i, l := range lines {
			if lipgloss.Width(l) > inner {
				t.Errorf("width %d: line %d is %d wide, the box holds %d", w, i, lipgloss.Width(l), inner)
			}
		}
		for i, l := range strings.Split(helpScreenView(a.width, canvasHeight(a), 0), "\n") {
			if lipgloss.Width(l) != width {
				t.Errorf("width %d: rendered line %d is %d wide, want %d", w, i, lipgloss.Width(l), width)
			}
		}
	}
}

func TestHelpFooterListsTheScrollKeys(t *testing.T) {
	a := helpApp(120, 30)
	a.helpOpen = true
	got := a.hintsView()
	for _, want := range []string{"close", "up", "down"} {
		if !strings.Contains(got, want) {
			t.Errorf("footer %q is missing %q", got, want)
		}
	}
}

func TestHelpThatFitsDoesNotScroll(t *testing.T) {
	a := helpApp(120, 100)
	if m := helpScrollMax(a.width, canvasHeight(a)); m != 0 {
		t.Fatalf("a tall terminal scrolls by %d", m)
	}
	if strings.Contains(helpScreenView(a.width, canvasHeight(a), 0), "%") {
		t.Error("a help screen that fits shows a scroll percentage")
	}
}
