package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/MawCeron/lazyftp/internal/shared"
)

type LogLevel = shared.LogLevel
type LogMsg = shared.LogMsg

const (
	LogInfo    = shared.LogInfo
	LogSuccess = shared.LogSuccess
	LogError   = shared.LogError
)

type LogEntry struct {
	Time    time.Time
	Message string
	Level   LogLevel
}

type LogPanel struct {
	entries []LogEntry
	maxSize int
}

func NewLogPanel() LogPanel {
	return LogPanel{maxSize: 100}
}

func (l LogPanel) Add(msg string, level LogLevel) LogPanel {
	l.entries = append(l.entries, LogEntry{
		Time:    time.Now(),
		Message: msg,
		Level:   level,
	})
	if len(l.entries) > l.maxSize {
		l.entries = l.entries[len(l.entries)-l.maxSize:]
	}
	return l
}

func (l LogPanel) Update(msg tea.Msg) (LogPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case LogMsg:
		return l.Add(msg.Message, msg.Level), nil
	}
	return l, nil
}

func (l LogPanel) View(width, height int) string {
	borderColor := lipgloss.Color("240")
	innerWidth := width - 4

	maxVisible := height - 3
	if maxVisible < 1 {
		maxVisible = 1
	}

	var rows []string
	start := 0
	if len(l.entries) > maxVisible {
		start = len(l.entries) - maxVisible
	}

	for _, e := range l.entries[start:] {
		rows = append(rows, renderLogEntry(e, innerWidth))
	}

	if len(l.entries) == 0 {
		rows = append(rows, lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("  (no logs)"))
	}

	body := strings.Join(rows, "\n")
	return borderWithTitle(body, "Log", width, height, borderColor)
}

func renderLogEntry(e LogEntry, width int) string {
	timeStr := e.Time.Format("15:04:05")

	color := lipgloss.Color("252")
	switch e.Level {
	case LogSuccess:
		color = lipgloss.Color("40")
	case LogError:
		color = lipgloss.Color("196")
	}

	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	msgStyle := lipgloss.NewStyle().Foreground(color)

	maxMsg := width - 12
	if maxMsg < 1 {
		maxMsg = 1
	}
	msg := e.Message
	if len(msg) > maxMsg {
		msg = msg[:maxMsg-3] + "..."
	}

	return fmt.Sprintf("  %s %s",
		timeStyle.Render("["+timeStr+"]"),
		msgStyle.Render(msg),
	)
}
