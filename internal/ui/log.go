package ui

import (
	"fmt"
	"io"
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
	file    io.Writer
}

// NewLogPanel writes everything it shows to file as well, if one is given. The
// panel keeps the last hundred entries; the file keeps the session.
func NewLogPanel(file io.Writer) LogPanel {
	return LogPanel{maxSize: 100, file: file}
}

var levelNames = map[LogLevel]string{
	LogInfo:    "INFO",
	LogSuccess: "OK",
	LogError:   "ERROR",
}

func (l LogPanel) Add(msg string, level LogLevel) LogPanel {
	entry := LogEntry{
		Time:    time.Now(),
		Message: msg,
		Level:   level,
	}

	l.entries = append(l.entries, entry)
	if len(l.entries) > l.maxSize {
		l.entries = l.entries[len(l.entries)-l.maxSize:]
	}

	if l.file != nil {
		// Written on the way in, unbuffered, so that a hang or a crash still
		// leaves everything that came before it on disk. The level is spelled
		// out because a file cannot carry the colour the panel uses.
		fmt.Fprintf(l.file, "%s %-5s %s\n",
			entry.Time.Format(time.RFC3339), levelNames[level], msg)
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
