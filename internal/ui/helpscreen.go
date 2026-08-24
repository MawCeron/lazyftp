package ui

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// helpScreenView renders the full key reference, grouped by context, from
// the exact same bindings keys.go declares for the footer -- so the two
// cannot list a key differently.
func helpScreenView(maxWidth, maxHeight int) string {
	hm := help.New()
	hm.Styles.FullKey = lipgloss.NewStyle().Bold(true).Foreground(colorEmphasis)
	hm.Styles.FullDesc = lipgloss.NewStyle().Foreground(colorMuted)
	hm.Styles.FullSeparator = lipgloss.NewStyle()

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(colorAccent)

	var sections []string
	for i, group := range helpGroups() {
		title := titleStyle.Render(helpGroupTitles[i])
		sections = append(sections, title+"\n"+hm.FullHelpView([][]key.Binding{group}))
	}
	body := strings.Join(sections, "\n\n")

	width := borderOuterWidth(lipgloss.Width(body))
	if width > maxWidth {
		width = maxWidth
	}
	height := lipgloss.Height(body) + 2
	if height > maxHeight {
		height = maxHeight
	}
	return borderWithTitle(body, "Help", width, height, colorAccent)
}
