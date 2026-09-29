package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const hostKeyContentWidth = 60

func hostKeyView(m hostKeyPromptMsg, maxWidth int) string {
	contentWidth := hostKeyContentWidth
	if outer := maxWidth - 4; outer < contentWidth {
		contentWidth = outer
	}
	if contentWidth < 1 {
		contentWidth = 1
	}

	labelStyle := lipgloss.NewStyle().Foreground(colorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(colorPrimary).Width(contentWidth)

	body := strings.Join([]string{
		lipgloss.NewStyle().Foreground(colorEmphasis).Width(contentWidth).Render("First time connecting to this host. Trust it?"),
		"",
		labelStyle.Render("Host:        ") + m.host,
		labelStyle.Render("Key type:    ") + m.keyType,
		labelStyle.Render("Fingerprint:"),
		valueStyle.Render(m.fingerprint),
	}, "\n")

	width := borderOuterWidth(lipgloss.Width(body))
	if width > maxWidth {
		width = maxWidth
	}
	height := lipgloss.Height(body) + 2
	return borderWithTitle(body, "Unknown Host", width, height, colorAccent)
}
