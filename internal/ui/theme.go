package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Semantic color tokens, defaulting to the dark palette (the common case)
// until SetTheme resolves the real background. Bubble Tea queries that
// asynchronously via tea.BackgroundColorMsg, so a static default avoids an
// unstyled first frame; app.go also falls back to this same dark default on
// a timeout, for terminals that never answer the query at all.
//
// Every value below was chosen against a representative dark (#1e1e1e) and
// light (#fafafa) background and checked for WCAG contrast: >=4.5:1 for
// anything that carries readable text, >=3:1 for colorBorder, which only
// draws structural lines and separators.
var (
	colorPrimary     color.Color = lipgloss.Color("#D4D4D4") // default text
	colorEmphasis    color.Color = lipgloss.Color("#F5F5F0") // footer key hints, "connected" detail
	colorMuted       color.Color = lipgloss.Color("#808592") // readable secondary text: dates, paths, placeholders
	colorBorder      color.Color = lipgloss.Color("#7A7F8C") // inactive borders, separators -- structure only
	colorAccent      color.Color = lipgloss.Color("#5DCAA5") // focus, cursor, active border
	colorSuccess     color.Color = lipgloss.Color("#97C459")
	colorError       color.Color = lipgloss.Color("#E8615F")
	colorDirectory   color.Color = lipgloss.Color("#85B7EB")
	colorMarked      color.Color = lipgloss.Color("#EFB050")
	colorBarBg       color.Color = lipgloss.Color("#282828") // status line / footer background
	colorDiffOnly    color.Color = lipgloss.Color("#E28560") // entries present on only one side (--highlight-diff)
	colorSizeDiffers color.Color = lipgloss.Color("#E080A0") // same name, different size (--highlight-diff)
)

// SetTheme resolves every token against the terminal's actual background.
// Called once from App.Update on tea.BackgroundColorMsg, or on the fallback
// timeout with isDark forced true.
func SetTheme(isDark bool) {
	ld := lipgloss.LightDark(isDark)
	colorPrimary = ld(lipgloss.Color("#2C2C2A"), lipgloss.Color("#D4D4D4"))
	colorEmphasis = ld(lipgloss.Color("#141414"), lipgloss.Color("#F5F5F0"))
	colorMuted = ld(lipgloss.Color("#74736A"), lipgloss.Color("#808592"))
	colorBorder = ld(lipgloss.Color("#8A897F"), lipgloss.Color("#7A7F8C"))
	colorAccent = ld(lipgloss.Color("#0F6E56"), lipgloss.Color("#5DCAA5"))
	colorSuccess = ld(lipgloss.Color("#3B6D11"), lipgloss.Color("#97C459"))
	colorError = ld(lipgloss.Color("#A32D2D"), lipgloss.Color("#E8615F"))
	colorDirectory = ld(lipgloss.Color("#185FA5"), lipgloss.Color("#85B7EB"))
	colorMarked = ld(lipgloss.Color("#854F0B"), lipgloss.Color("#EFB050"))
	colorBarBg = ld(lipgloss.Color("#EFEDE6"), lipgloss.Color("#282828"))
	colorDiffOnly = ld(lipgloss.Color("#993C1D"), lipgloss.Color("#E28560"))
	colorSizeDiffers = ld(lipgloss.Color("#993556"), lipgloss.Color("#E080A0"))
}
