package ui

import "github.com/charmbracelet/lipgloss"

// Semantic color tokens. Values are the AdaptiveColor's Dark/Light 256-color
// codes; lipgloss/termenv downgrade automatically to ANSI16 or monochrome
// on terminals that support less, and honor NO_COLOR.
var (
	colorAccent    = lipgloss.AdaptiveColor{Light: "25", Dark: "39"}   // focus, cursor
	colorMuted     = lipgloss.AdaptiveColor{Light: "240", Dark: "240"} // secondary text, inactive border, separators
	colorPrimary   = lipgloss.AdaptiveColor{Light: "235", Dark: "252"} // default text
	colorEmphasis  = lipgloss.AdaptiveColor{Light: "235", Dark: "255"} // footer key hints
	colorSuccess   = lipgloss.AdaptiveColor{Light: "28", Dark: "40"}
	colorError     = lipgloss.AdaptiveColor{Light: "160", Dark: "196"}
	colorDirectory = lipgloss.AdaptiveColor{Light: "26", Dark: "75"}
	colorMarked    = lipgloss.AdaptiveColor{Light: "130", Dark: "214"}
)
