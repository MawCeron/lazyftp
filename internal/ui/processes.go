package ui

import (
	"fmt"
	"strings"

	"github.com/MawCeron/lazyftp/internal/shared"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TransferDirection = shared.TransferDirection
type TransferStatus = shared.TransferStatus
type Transfer = shared.Transfer
type TransferStartMsg = shared.TransferStartMsg
type TransferProgressMsg = shared.TransferProgressMsg
type TransferErrorMsg = shared.TransferErrorMsg

const (
	DirectionUpload   = shared.DirectionUpload
	DirectionDownload = shared.DirectionDownload
	StatusInProgress  = shared.StatusInProgress
	StatusDone        = shared.StatusDone
	StatusError       = shared.StatusError
)

type ProcessesPanel struct {
	transfers []Transfer
}

func NewProcessesPanel() ProcessesPanel {
	return ProcessesPanel{}
}

func (p ProcessesPanel) AddTransfer(t Transfer) ProcessesPanel {
	p.transfers = append(p.transfers, t)
	return p
}

func (p ProcessesPanel) UpdateTransfer(filename string, current int64) ProcessesPanel {
	for i, t := range p.transfers {
		if t.Filename == filename {
			p.transfers[i].Current = current
			if current >= t.Total && t.Total > 0 {
				p.transfers[i].Status = StatusDone
			}
		}
	}
	return p
}

func (p ProcessesPanel) MarkError(filename string) ProcessesPanel {
	for i, t := range p.transfers {
		if t.Filename == filename {
			p.transfers[i].Status = StatusError
		}
	}
	return p
}

func (p ProcessesPanel) Update(msg tea.Msg) (ProcessesPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case TransferStartMsg:
		return p.AddTransfer(msg.Transfer), nil
	case TransferProgressMsg:
		return p.UpdateTransfer(msg.Filename, msg.Current), nil
	case TransferErrorMsg:
		return p.MarkError(msg.Filename), nil
	}
	return p, nil
}

func (p ProcessesPanel) View(width, height int) string {
	borderColor := lipgloss.Color("240")
	innerWidth := width - 4

	visibleHeight := height - 3
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	var rows []string
	start := 0
	if len(p.transfers) > visibleHeight {
		start = len(p.transfers) - visibleHeight
	}
	for _, t := range p.transfers[start:] {
		rows = append(rows, renderTransfer(t, innerWidth))
	}

	if len(p.transfers) == 0 {
		rows = append(rows, lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("  (no transfers)"))
	}

	body := strings.Join(rows, "\n\n")
	return borderWithTitle(body, "Processes", width, height, borderColor)
}

func renderTransfer(t Transfer, width int) string {
	dirSymbol := "↑"
	if t.Direction == DirectionDownload {
		dirSymbol = "↓"
	}

	name := t.Filename
	maxName := 20
	if len(name) > maxName {
		name = name[:maxName-3] + "..."
	}

	barWidth := width - maxName - 12
	if barWidth < 8 {
		barWidth = 8
	}

	progress := t.Progress()
	filled := int(float64(barWidth) * progress)

	bar := "[" + strings.Repeat("█", filled) +
		strings.Repeat("░", barWidth-filled) + "]"

	suffix := fmt.Sprintf(" %d%%  %s", int(progress*100), dirSymbol)
	if t.Status == StatusDone {
		suffix = " ✔"
		bar = lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Render(bar)
	} else if t.Status == StatusError {
		suffix = " ✗"
		bar = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(bar)
	}

	return fmt.Sprintf("  %-*s %s%s", maxName, name, bar, suffix)
}
