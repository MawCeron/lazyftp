package ui

import (
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MawCeron/lazyftp/internal/theme"
)

func resetTheme(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { activeTheme = nil; SetTheme(true) })
	activeTheme = nil
	SetTheme(true)
}

func newThemedApp(th *theme.Theme) App {
	return NewApp(nil, false, nil, "dev", false).WithTheme(th, nil)
}

var (
	lightBg = tea.BackgroundColorMsg{Color: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}}
	darkBg  = tea.BackgroundColorMsg{Color: color.RGBA{A: 0xff}}
)

func TestAThemeOverridesOnlyTheTokensItNames(t *testing.T) {
	resetTheme(t)
	activeTheme = &theme.Theme{Dark: theme.Colors{Accent: "#123456"}}

	SetTheme(true)
	if colorAccent != lipgloss.Color("#123456") {
		t.Error("the named token was not overridden")
	}
	if colorPrimary != lipgloss.Color("#D4D4D4") {
		t.Error("a token the theme leaves out lost its built-in color")
	}

	SetTheme(false)
	if colorAccent != lipgloss.Color("#0F6E56") {
		t.Error("the palette the theme does not define should stay built-in")
	}
}

func TestShortHexColorsWork(t *testing.T) {
	r, g, b, _ := lipgloss.Color("#fff").RGBA()
	if r != 0xffff || g != 0xffff || b != 0xffff {
		t.Fatalf("#fff parsed as %x %x %x", r, g, b)
	}
}

func TestABackgroundInOnePaletteFixesThePaletteAndStopsDetecting(t *testing.T) {
	resetTheme(t)
	a := newThemedApp(&theme.Theme{Dark: theme.Colors{Background: "#0000AA", Primary: "#FFFF55"}})

	if !a.themeFixed || colorBackground != lipgloss.Color("#0000AA") || colorPrimary != lipgloss.Color("#FFFF55") {
		t.Fatalf("fixed=%v background=%v", a.themeFixed, colorBackground)
	}
	if n := len(a.detectionCmds()); n != 0 {
		t.Errorf("%d detection commands still run", n)
	}

	// Whatever the terminal says, including a light background, changes nothing.
	got, _ := a.Update(lightBg)
	got, _ = got.(App).Update(themeFallbackMsg{})
	if colorBackground != lipgloss.Color("#0000AA") || colorPrimary != lipgloss.Color("#FFFF55") {
		t.Error("a background report from the terminal overrode the theme")
	}
	if _, cmd := got.(App).Update(themePollMsg{}); cmd != nil {
		t.Error("the poll kept itself going")
	}
}

func TestALightOnlyBackgroundPicksTheLightPalette(t *testing.T) {
	resetTheme(t)
	a := newThemedApp(&theme.Theme{Light: theme.Colors{Background: "#FDF6E3"}})
	if !a.themeFixed || colorBackground != lipgloss.Color("#FDF6E3") || colorPrimary != lipgloss.Color("#2C2C2A") {
		t.Fatalf("fixed=%v background=%v primary=%v", a.themeFixed, colorBackground, colorPrimary)
	}
}

func TestBackgroundsInBothPalettesAskTheTerminalExactlyOnce(t *testing.T) {
	resetTheme(t)
	a := newThemedApp(&theme.Theme{
		Dark:  theme.Colors{Background: "#1E1E2E"},
		Light: theme.Colors{Background: "#EFF1F5"},
	})
	if a.themeFixed || !a.themeOnce {
		t.Fatalf("fixed=%v once=%v", a.themeFixed, a.themeOnce)
	}
	if n := len(a.detectionCmds()); n != 2 {
		t.Errorf("%d detection commands, want the query and its timeout but no poll", n)
	}

	got, _ := a.Update(lightBg)
	a = got.(App)
	if colorBackground != lipgloss.Color("#EFF1F5") || !a.themeFixed {
		t.Fatalf("light terminal: background %v, fixed %v", colorBackground, a.themeFixed)
	}
	if n := len(a.detectionCmds()); n != 0 {
		t.Errorf("%d detection commands after the choice", n)
	}

	// Our own background change would come back as a report; it must not flip it.
	a.Update(darkBg)
	if colorBackground != lipgloss.Color("#EFF1F5") {
		t.Error("a later report flipped the palette")
	}
}

func TestAThemeWithoutABackgroundKeepsDetectionRunning(t *testing.T) {
	resetTheme(t)
	a := newThemedApp(&theme.Theme{Dark: theme.Colors{Accent: "#123456"}, Light: theme.Colors{Accent: "#654321"}})
	if a.themeFixed || a.themeOnce || len(a.detectionCmds()) != 3 {
		t.Fatalf("fixed=%v once=%v cmds=%d", a.themeFixed, a.themeOnce, len(a.detectionCmds()))
	}
	a.Update(lightBg)
	if colorAccent != lipgloss.Color("#654321") {
		t.Error("the light palette was not picked for a light terminal")
	}
}

func TestTheBackgroundReachesTheTerminalUnlessNoColorIsSet(t *testing.T) {
	resetTheme(t)
	a := newThemedApp(&theme.Theme{Dark: theme.Colors{Background: "#0000AA"}})
	t.Setenv("NO_COLOR", "")
	if got := a.View().BackgroundColor; got != lipgloss.Color("#0000AA") {
		t.Errorf("View background %v", got)
	}
	t.Setenv("NO_COLOR", "1")
	if got := a.View().BackgroundColor; got != nil {
		t.Errorf("NO_COLOR set but View asks for %v", got)
	}

	resetTheme(t)
	t.Setenv("NO_COLOR", "")
	if got := NewApp(nil, false, nil, "dev", false).View().BackgroundColor; got != nil {
		t.Errorf("no theme, yet View asks for %v", got)
	}
}

func TestAThemeThatFailedToLoadIsALogLineAndNothingElse(t *testing.T) {
	resetTheme(t)
	a := NewApp(nil, false, nil, "dev", false).WithTheme(nil, errTest("theme \"x\" not found"))
	if got := lastLog(a); got != "Using the default theme: theme \"x\" not found" {
		t.Errorf("log %q", got)
	}
	if a.themeFixed || colorBackground != nil {
		t.Error("a failed theme changed the colors")
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }
