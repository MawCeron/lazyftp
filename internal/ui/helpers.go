package ui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

func borderWithTitle(content, title string, width, height int, borderColor color.Color) string {
	lines := strings.Split(content, "\n")
	maxLines := height - 4
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:] // takes the last maxLines lines
		content = strings.Join(lines, "\n")
	}

	titleStr := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent).
		MarginBottom(1).
		Render(title)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width-2).
		Height(height-2).
		Padding(0, 1).
		Render(titleStr + "\n" + content)
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
