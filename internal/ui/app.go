package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focus int

const (
	focusLocal focus = iota
	focusRemote
	focusConnectionBar
)

type App struct {
	width  int
	height int
	focus  focus

	connBar   ConnectionBar
	local     Panel
	remote    Panel
	processes ProcessesPanel
	log       LogPanel
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q":
			return a, tea.Quit
		case "ctrl+l":
			a.focus = focusConnectionBar
			return a, nil
		case "tab":
			if a.focus == focusLocal {
				a.focus = focusRemote
			} else {
				a.focus = focusLocal
			}
			return a, nil
		case "esc":
			if a.focus == focusConnectionBar {
				a.focus = focusLocal
			}
			return a, nil
		}
	}

	// Delegate to focused component
	switch a.focus {
	case focusConnectionBar:
		a.connBar, _ = a.connBar.Update(msg)
	case focusLocal:
		a.local, _ = a.local.Update(msg)
	case focusRemote:
		a.remote, _ = a.remote.Update(msg)
	}
	return a, nil
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	top := a.connBar.View(a.width, a.focus == focusConnectionBar)

	panelWidth := a.width / 2
	localView := a.local.View(panelWidth, a.height/2, a.focus == focusLocal)
	remoteView := a.local.View(panelWidth, a.height/2, a.focus == focusRemote)
	panels := lipgloss.JoinHorizontal(lipgloss.Top, localView, remoteView)

	bottomWidth := a.width / 2
	processesView := a.processes.View(bottomWidth)
	logView := a.log.View(bottomWidth)
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, processesView, logView)

	return lipgloss.JoinVertical(lipgloss.Left, top, panels, bottom)
}
