package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MawCeron/lazyftp/internal/shared"
	"github.com/mattn/go-runewidth"
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

// The panel drops old entries; the file keeps the whole session.
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
		// Unbuffered, so a crash still leaves what came before it on disk.
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
	borderColor := colorMuted
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
			Foreground(colorMuted).
			Render("  (no logs)"))
	}

	body := strings.Join(rows, "\n")
	return borderWithTitle(body, "Log", width, height, borderColor)
}

func renderLogEntry(e LogEntry, width int) string {
	timeStr := e.Time.Format("15:04:05")

	color := colorPrimary
	switch e.Level {
	case LogSuccess:
		color = colorSuccess
	case LogError:
		color = colorError
	}

	timeStyle := lipgloss.NewStyle().Foreground(colorMuted)
	levelStyle := lipgloss.NewStyle().Foreground(color)
	msgStyle := lipgloss.NewStyle().Foreground(color)

	level := fmt.Sprintf("[%-5s]", levelNames[e.Level])

	maxMsg := width - 21
	if maxMsg < 1 {
		maxMsg = 1
	}
	msg := runewidth.Truncate(e.Message, maxMsg, "...")

	return fmt.Sprintf("  %s %s %s",
		timeStyle.Render("["+timeStr+"]"),
		levelStyle.Render(level),
		msgStyle.Render(msg),
	)
}
