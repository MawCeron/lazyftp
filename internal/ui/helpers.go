package ui

import "github.com/charmbracelet/lipgloss"

func borderWithTitle(content, title string, width, height int, borderColor lipgloss.Color) string {
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
