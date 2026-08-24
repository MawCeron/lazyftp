package ui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

// borderWithTitle draws a rounded box with the title set into the top border
// itself, tmux/lazygit-style, instead of on its own line above the content —
// that line and the blank line separating it from the content were the two
// rows this used to cost every panel before the content even started.
func borderWithTitle(content, title string, width, height int, borderColor color.Color) string {
	lines := strings.Split(content, "\n")
	maxLines := height - 2
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:] // takes the last maxLines lines
		content = strings.Join(lines, "\n")
	}

	border := lipgloss.RoundedBorder()
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(colorAccent)

	// Matches the box's own Width(boxWidth) below: Lipgloss renders every
	// line of a bordered, padded block at that total width, corners
	// included, so the hand-built top row has to target the same number to
	// close flush with the corners instead of overshooting them.
	boxWidth := width - 2
	label := " " + title + " "
	fill := boxWidth - 3 - runewidth.StringWidth(label) // 2 corners + the leading dash
	if fill < 0 {
		fill = 0
	}
	top := borderStyle.Render(border.TopLeft+border.Top) +
		titleStyle.Render(label) +
		borderStyle.Render(strings.Repeat(border.Top, fill)+border.TopRight)

	box := lipgloss.NewStyle().
		Border(border).
		BorderTop(false).
		BorderForeground(borderColor).
		Width(boxWidth).
		Height(height-2).
		Padding(0, 1).
		Render(content)

	return top + "\n" + box
}

// borderInteriorWidth returns how many cells of a borderWithTitle box at the
// given outer width are actually usable for content, once its border and
// padding (1 cell each side, both) are subtracted.
func borderInteriorWidth(width int) int {
	return width - 6
}

// borderOuterWidth is borderInteriorWidth's inverse: the outer width a
// borderWithTitle box needs so that contentWidth cells of content fit
// without wrapping.
func borderOuterWidth(contentWidth int) int {
	return contentWidth + 6
}

// truncateHead keeps the trailing portion of s that fits within width cells,
// prefixing "…" when s had to be cut. Used for paths, where the leaf at the
// end matters more than the root.
func truncateHead(s string, width int) string {
	if runewidth.StringWidth(s) <= width {
		return s
	}

	const ellipsis = "…"
	budget := width - runewidth.StringWidth(ellipsis)

	runes := []rune(s)
	w, start := 0, len(runes)
	for i := len(runes) - 1; i >= 0 && w < budget; i-- {
		rw := runewidth.RuneWidth(runes[i])
		if w+rw > budget {
			break
		}
		w += rw
		start = i
	}

	return ellipsis + string(runes[start:])
}

// formatSize renders a byte count with binary units, one decimal place past
// the first: "512 B", "45.2 MB", "1.3 GB".
func formatSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
