package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func borderWithTitle(content, title string, width, height int, borderColor lipgloss.Color) string {
	lines := strings.Split(content, "\n")
	maxLines := height - 4
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:] // takes the last maxLines lines
		content = strings.Join(lines, "\n")
	}

	titleStr := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
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
