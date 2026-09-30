package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// helpScreenLayout renders the full key reference, grouped by context, from
// the exact same bindings keys.go declares for the footer -- so the two
// cannot list a key differently. The text is wrapped to the box's interior
// width and returned as lines, since the box shows a window onto them.
func helpScreenLayout(maxWidth, maxHeight int) (width, height int, lines []string) {
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

	width = min(borderOuterWidth(lipgloss.Width(body)), max(maxWidth, 5))
	contentWidth := max(borderInteriorWidth(width), 1)
	wrapped := lipgloss.NewStyle().Width(contentWidth).Render(body)
	lines = strings.Split(wrapped, "\n")
	height = min(len(lines)+2, max(maxHeight, 3))
	return width, height, lines
}

// helpScrollMax is how far down the help text can be scrolled: zero when it all
// fits.
func helpScrollMax(maxWidth, maxHeight int) int {
	_, height, lines := helpScreenLayout(maxWidth, maxHeight)
	return max(0, len(lines)-(height-2))
}

func helpScreenView(maxWidth, maxHeight, offset int) string {
	width, height, lines := helpScreenLayout(maxWidth, maxHeight)
	visible := height - 2
	offset = max(0, min(offset, len(lines)-visible))

	title := "Help"
	if len(lines) > visible {
		title = fmt.Sprintf("Help %d%%", (offset+visible)*100/len(lines))
	}
	return borderWithTitle(strings.Join(lines[offset:offset+visible], "\n"), title, width, height, colorAccent)
}
