package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
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
	entries  []LogEntry
	maxSize  int
	file     io.Writer
	viewport viewport.Model
}

// The panel drops old entries; the file keeps the whole session.
func NewLogPanel(file io.Writer) LogPanel {
	return LogPanel{maxSize: 100, file: file, viewport: viewport.New()}
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

	// Only follows the new entry down if the view was already at the
	// bottom -- scrolled up reading history, a new line arriving shouldn't
	// yank the view out from under you.
	atBottom := l.viewport.AtBottom()
	l.viewport.SetContent(l.renderEntries())
	if atBottom {
		l.viewport.GotoBottom()
	}

	return l
}

// SetSize persists the viewport's dimensions -- unlike Panel's list, a
// viewport doesn't own the space it's given until told to, and View()
// itself only returns a string, with no way to hand size changes back for
// AtBottom's math in Add to stay accurate.
func (l LogPanel) SetSize(width, height int) LogPanel {
	innerWidth := borderInteriorWidth(width)
	if innerWidth < 1 {
		innerWidth = 1
	}
	contentHeight := height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}
	l.viewport.SetWidth(innerWidth)
	l.viewport.SetHeight(contentHeight)
	l.viewport.SetContent(l.renderEntries())
	return l
}

// UpdateFocused scrolls the viewport. Only called while the Log panel has
// focus -- unlike Update below, which always runs so new entries keep
// arriving regardless of what's focused.
func (l LogPanel) UpdateFocused(msg tea.Msg) (LogPanel, tea.Cmd) {
	var cmd tea.Cmd
	l.viewport, cmd = l.viewport.Update(msg)
	return l, cmd
}

func (l LogPanel) Update(msg tea.Msg) (LogPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case LogMsg:
		return l.Add(msg.Message, msg.Level), nil
	}
	return l, nil
}

func (l LogPanel) View(width, height int, active bool) string {
	borderColor := colorBorder
	if active {
		borderColor = colorAccent
	}
	return borderWithTitle(l.viewport.View(), "Log", width, height, borderColor)
}

func (l LogPanel) renderEntries() string {
	if len(l.entries) == 0 {
		return lipgloss.NewStyle().Foreground(colorMuted).Render("  (no logs)")
	}
	innerWidth := l.viewport.Width()
	rows := make([]string, len(l.entries))
	for i, e := range l.entries {
		rows[i] = renderLogEntry(e, innerWidth)
	}
	return strings.Join(rows, "\n")
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
