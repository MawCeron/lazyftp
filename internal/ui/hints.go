package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type hintItem struct {
	key   string
	label string
}

var hintsConnectionBar = []hintItem{
	{"tab", "next field"},
	{"shift+tab", "prev field"},
	{"enter", "connect"},
	{"esc", "dismiss"},
}

var hintsLocal = []hintItem{
	{"tab", "switch panel"},
	{"enter/space", "open dir"},
	{"-", "up"},
	{"x", "mark"},
	{"t", "upload"},
	{"ctrl+l", "connection"},
	{"q", "quit"},
}

var hintsRemote = []hintItem{
	{"tab", "switch panel"},
	{"enter/space", "open dir"},
	{"-", "up"},
	{"x", "mark"},
	{"t", "download"},
	{"ctrl+l", "connection"},
	{"q", "quit"},
}

func renderHints(f focus, width int) string {
	var items []hintItem
	switch f {
	case focusConnectionBar:
		items = hintsConnectionBar
	case focusLocal:
		items = hintsLocal
	case focusRemote:
		items = hintsRemote
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	var parts []string
	for _, h := range items {
		part := keyStyle.Render(h.key) + " " + labelStyle.Render(h.label)
		parts = append(parts, part)
	}

	bar := strings.Join(parts, sepStyle.Render("  |  "))

	return lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Render(bar)
}
