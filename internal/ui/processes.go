package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MawCeron/lazyftp/internal/shared"
	"github.com/mattn/go-runewidth"
)

type ProcessesPanel struct {
	transfers []shared.Transfer
}

func NewProcessesPanel() ProcessesPanel {
	return ProcessesPanel{}
}

func (p ProcessesPanel) AddTransfer(t shared.Transfer) ProcessesPanel {
	p.transfers = append(p.transfers, t)
	return p
}

func (p ProcessesPanel) UpdateTransfer(filename string, current int64) ProcessesPanel {
	for i, t := range p.transfers {
		if t.Filename == filename {
			p.transfers[i].Current = current
			if current >= t.Total && t.Total > 0 {
				p.transfers[i].Status = shared.StatusDone
			}
		}
	}
	return p
}

func (p ProcessesPanel) MarkError(filename string) ProcessesPanel {
	for i, t := range p.transfers {
		if t.Filename == filename {
			p.transfers[i].Status = shared.StatusError
		}
	}
	return p
}

func (p ProcessesPanel) Update(msg tea.Msg) (ProcessesPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.TransferStartMsg:
		return p.AddTransfer(msg.Transfer), nil
	case shared.TransferProgressMsg:
		return p.UpdateTransfer(msg.Filename, msg.Current), nil
	case shared.TransferErrorMsg:
		return p.MarkError(msg.Filename), nil
	}
	return p, nil
}

func (p ProcessesPanel) View(width, height int) string {
	borderColor := colorMuted
	innerWidth := borderInteriorWidth(width)

	visibleHeight := (height - 2) / 2 // Each transfer takes 2 lines, and the border reserves 2
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
			Foreground(colorMuted).
			Render("  (no transfers)"))
	}

	body := strings.Join(rows, "\n")
	return borderWithTitle(body, "Processes", width, height, borderColor)
}

func renderTransfer(t shared.Transfer, width int) string {
	dirSymbol := iconUpload()
	if t.Direction == shared.DirectionDownload {
		dirSymbol = iconDownload()
	}

	maxName := 20
	name := runewidth.Truncate(t.Filename, maxName, "...")
	name = runewidth.FillRight(name, maxName)

	barWidth := width - maxName - 12
	if barWidth < 4 {
		barWidth = 4
	}

	progress := t.Progress()
	filled := int(float64(barWidth) * progress)
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}

	bar := "[" + strings.Repeat("█", filled) +
		strings.Repeat("░", barWidth-filled) + "]"

	suffix := fmt.Sprintf(" %d%%  %s", int(progress*100), dirSymbol)
	switch t.Status {
	case shared.StatusDone:
		suffix = " " + iconDone()
		bar = lipgloss.NewStyle().Foreground(colorSuccess).Render(bar)
	case shared.StatusError:
		suffix = " " + iconError()
		bar = lipgloss.NewStyle().Foreground(colorError).Render(bar)
	}

	return fmt.Sprintf("  %s %s%s\n", name, bar, suffix)
}
