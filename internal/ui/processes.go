package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MawCeron/lazyftp/internal/shared"
	"github.com/mattn/go-runewidth"
)

type ProcessesPanel struct {
	transfers []shared.Transfer
	viewport  viewport.Model
}

func NewProcessesPanel() ProcessesPanel {
	return ProcessesPanel{viewport: viewport.New()}
}

func (p ProcessesPanel) AddTransfer(t shared.Transfer) ProcessesPanel {
	p.transfers = append(p.transfers, t)
	return p.refreshViewport()
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
	return p.refreshViewport()
}

func (p ProcessesPanel) MarkError(filename string) ProcessesPanel {
	for i, t := range p.transfers {
		if t.Filename == filename {
			p.transfers[i].Status = shared.StatusError
		}
	}
	return p.refreshViewport()
}

// refreshViewport re-renders the transfer list into the viewport, following
// the newest entry down only if the view was already caught up -- the same
// rule LogPanel's Add uses, so reading an in-progress transfer's history
// isn't yanked away by the next progress tick.
func (p ProcessesPanel) refreshViewport() ProcessesPanel {
	atBottom := p.viewport.AtBottom()
	p.viewport.SetContent(p.renderTransfers())
	if atBottom {
		p.viewport.GotoBottom()
	}
	return p
}

// SetSize persists the viewport's dimensions; see LogPanel.SetSize for why
// that has to be explicit rather than computed fresh in View.
func (p ProcessesPanel) SetSize(width, height int) ProcessesPanel {
	innerWidth := borderInteriorWidth(width)
	if innerWidth < 1 {
		innerWidth = 1
	}
	contentHeight := height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}
	p.viewport.SetWidth(innerWidth)
	p.viewport.SetHeight(contentHeight)
	p.viewport.SetContent(p.renderTransfers())
	return p
}

// UpdateFocused scrolls the viewport. Only called while Processes has focus;
// Update below always runs so progress keeps arriving regardless of focus.
func (p ProcessesPanel) UpdateFocused(msg tea.Msg) (ProcessesPanel, tea.Cmd) {
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
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

func (p ProcessesPanel) View(width, height int, active bool) string {
	borderColor := colorBorder
	if active {
		borderColor = colorAccent
	}
	return borderWithTitle(p.viewport.View(), "Processes", width, height, borderColor)
}

func (p ProcessesPanel) renderTransfers() string {
	if len(p.transfers) == 0 {
		return lipgloss.NewStyle().Foreground(colorMuted).Render("  (no transfers)")
	}
	innerWidth := p.viewport.Width()
	rows := make([]string, len(p.transfers))
	for i, t := range p.transfers {
		rows[i] = renderTransfer(t, innerWidth)
	}
	return strings.Join(rows, "\n")
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
