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
	contentWidth := modalContentWidth(fileInfoContentWidth, maxWidth)

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

	return modalBox("File Info", body, maxWidth)
}
