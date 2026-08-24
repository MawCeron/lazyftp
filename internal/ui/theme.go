package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Semantic color tokens, defaulting to the values that suit a dark terminal
// (the common case) until SetTheme resolves the real background. Bubble Tea
// queries that asynchronously via tea.BackgroundColorMsg, so a static
// default avoids an unstyled first frame.
var (
	colorAccent    color.Color = lipgloss.Color("39")  // focus, cursor
	colorMuted     color.Color = lipgloss.Color("240") // secondary text, inactive border, separators
	colorPrimary   color.Color = lipgloss.Color("252") // default text
	colorEmphasis  color.Color = lipgloss.Color("255") // footer key hints
	colorSuccess   color.Color = lipgloss.Color("40")
	colorError     color.Color = lipgloss.Color("196")
	colorDirectory color.Color = lipgloss.Color("75")
	colorMarked    color.Color = lipgloss.Color("214")
	colorBarBg     color.Color = lipgloss.Color("235") // status line / footer background
	colorDiffOnly  color.Color = lipgloss.Color("226") // entries present on only one side (--highlight-diff)
)

// SetTheme resolves every token against the terminal's actual background.
// Call once from App.Update on tea.BackgroundColorMsg.
func SetTheme(isDark bool) {
	ld := lipgloss.LightDark(isDark)
	colorAccent = ld(lipgloss.Color("25"), lipgloss.Color("39"))
	colorMuted = ld(lipgloss.Color("240"), lipgloss.Color("240"))
	colorPrimary = ld(lipgloss.Color("235"), lipgloss.Color("252"))
	colorEmphasis = ld(lipgloss.Color("235"), lipgloss.Color("255"))
	colorSuccess = ld(lipgloss.Color("28"), lipgloss.Color("40"))
	colorError = ld(lipgloss.Color("160"), lipgloss.Color("196"))
	colorDirectory = ld(lipgloss.Color("26"), lipgloss.Color("75"))
	colorMarked = ld(lipgloss.Color("130"), lipgloss.Color("214"))
	colorBarBg = ld(lipgloss.Color("253"), lipgloss.Color("235"))
	colorDiffOnly = ld(lipgloss.Color("136"), lipgloss.Color("226"))
}
