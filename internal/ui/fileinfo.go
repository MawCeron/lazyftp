package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/mattn/go-runewidth"
)

// fileInfoContentWidth caps the box at a tidy dialog width regardless of how
// wide the terminal is -- a single long filename has no reason to stretch it
// across the whole screen.
const fileInfoContentWidth = 56

// fileInfoView renders the exact size and full-precision timestamp a narrow
// panel drops to make room for the name. (#71)
func fileInfoView(file model.FileInfo, maxWidth int) string {
	contentWidth := fileInfoContentWidth
	if outer := maxWidth - 4; outer < contentWidth {
		contentWidth = outer
	}
	if contentWidth < 1 {
		contentWidth = 1
	}

	name := runewidth.Truncate(file.Name, contentWidth, "...")

	labelStyle := lipgloss.NewStyle().Foreground(colorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(colorPrimary)

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(colorEmphasis).Render(name),
		"",
		labelStyle.Render("Size:     ") + valueStyle.Render(fmt.Sprintf("%s (%d bytes)", formatSize(file.Size), file.Size)),
		labelStyle.Render("Modified: ") + valueStyle.Render(file.ModTime.Format("2006-01-02 15:04:05")),
	}
	body := strings.Join(lines, "\n")

	width := borderOuterWidth(lipgloss.Width(body))
	if width > maxWidth {
		width = maxWidth
	}
	height := lipgloss.Height(body) + 2
	return borderWithTitle(body, "File Info", width, height, colorAccent)
}
