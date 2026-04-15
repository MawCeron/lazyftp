package ui

import (
	"os"
	"strconv"
	"strings"

	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
	"github.com/MawCeron/lazyftp/internal/transfer"
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

	client  client.Client
	manager *transfer.Manager
	program func() *tea.Program

	connBar   ConnectionBar
	local     Panel
	remote    Panel
	processes ProcessesPanel
	log       LogPanel

	connected bool
}

func NewApp(p func() *tea.Program) App {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}

	app := App{
		focus:     focusConnectionBar,
		connBar:   NewConnectionBar(),
		local:     NewPanel("LOCAL"),
		remote:    NewPanel("REMOTE"),
		processes: NewProcessesPanel(),
		log:       NewLogPanel(),
		program:   p,
	}
	app.local.path = home
	return app
}

func (a App) Init() tea.Cmd {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}
	return loadLocalDir(home)
}

// heights returns calculated heights of each section
func (a App) heights() (connH, panelH, bottomH int) {
	connH = 5    // ConnectionBar field
	bottomH = 10 // Processes + Log minimal fixed
	hintsH := 1
	panelH = a.height - connH - bottomH - hintsH
	if panelH < 8 {
		panelH = 8
	}
	// recalculate boottom with real space
	bottomH = a.height - connH - panelH - hintsH
	if bottomH < 8 {
		bottomH = 8
	}
	return
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		_, panelH, _ := a.heights()
		panelW := a.width / 2
		a.local = a.local.SetSize(panelW, panelH)
		a.remote = a.remote.SetSize(panelW, panelH)
		return a, nil

	case tea.KeyMsg:
		if a.focus != focusConnectionBar {
			switch msg.String() {
			case "q", "Q":
				if a.client != nil {
					a.client.Disconnect()
				}
				return a, tea.Quit
			}
		}

		switch msg.String() {
		case "ctrl+l":
			a.focus = focusConnectionBar
			return a, nil
		case "tab":
			if a.focus != focusConnectionBar {
				if a.focus == focusLocal {
					a.focus = focusRemote
				} else {
					a.focus = focusLocal
				}
				return a, nil
			}
		case "esc":
			if a.focus == focusConnectionBar {
				a.focus = focusLocal
			}
			return a, nil
		}

	case ConnectMsg:
		return a.handleConnect(msg)

	case NavigateMsg:
		return a.handleNavigate(msg)

	case TransferMsg:
		return a.handleTransfer(msg)

	case LocalDirLoadedMsg:
		a.local = a.local.WithFiles(msg.Files, msg.Path)
		_, panelH, _ := a.heights()
		a.local = a.local.SetSize(a.width/2, panelH)
		return a, nil

	case RemoteDirLoadedMsg:
		a.remote = a.remote.WithFiles(msg.Files, msg.Path)
		_, panelH, _ := a.heights()
		a.remote = a.remote.SetSize(a.width/2, panelH)
		return a, nil

	case TransferDoneMsg:
		return a.handleTransferDone(msg)
	}

	// processes and log are always listening
	a.processes, _ = a.processes.Update(msg)
	a.log, _ = a.log.Update(msg)

	switch a.focus {
	case focusConnectionBar:
		a.connBar, cmd = a.connBar.Update(msg)
	case focusLocal:
		a.local, cmd = a.local.Update(msg)
	case focusRemote:
		a.remote, cmd = a.remote.Update(msg)
	}

	return a, cmd
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	_, panelH, bottomH := a.heights()
	panelW := a.width / 2

	top := a.connBar.View(a.width, a.focus == focusConnectionBar)
	localView := a.local.View(panelW, panelH, a.focus == focusLocal)
	remoteView := a.remote.View(panelW, panelH, a.focus == focusRemote)
	panels := lipgloss.JoinHorizontal(lipgloss.Top, localView, remoteView)

	processesView := a.processes.View(panelW, bottomH)
	logView := a.log.View(panelW, bottomH)
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, processesView, logView)

	hints := a.hintsView()

	return lipgloss.JoinVertical(lipgloss.Left, top, panels, bottom, hints)
}

func (a App) hintsView() string {
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	sep := sepStyle.Render(" | ")

	hint := func(key, desc string) string {
		return keyStyle.Render(key) + ": " + descStyle.Render(desc)
	}

	var hints []string
	switch a.focus {
	case focusConnectionBar:
		hints = []string{
			hint("Tab", "next field"),
			hint("Shift+Tab", "prev field"),
			hint("Enter", "connect"),
			hint("Esc", "close"),
		}
	case focusLocal, focusRemote:
		hints = []string{
			hint("j/k", "navigate"),
			hint("Enter", "open dir"),
			hint("-", "go up"),
			hint("x", "mark"),
			hint("t", "transfer"),
			hint("Tab", "switch panel"),
			hint("Ctrl+L", "connection"),
			hint("q", "quit"),
		}
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(a.width).
		Render(strings.Join(hints, sep))
}

// handlers

func (a App) handleConnect(msg ConnectMsg) (App, tea.Cmd) {
	port, err := strconv.Atoi(msg.Port)
	if err != nil || port <= 0 {
		port = 22
	}

	if a.client != nil {
		a.client.Disconnect()
	}

	var c client.Client
	if port == 22 {
		c = client.NewSFTPClient()
	} else {
		c = client.NewFTPClient()
	}

	if err := c.Connect(msg.Host, msg.User, msg.Pass, port); err != nil {
		a.log = a.log.Add("Error connecting: "+err.Error(), LogError)
		return a, nil
	}

	a.client = c
	a.manager = transfer.NewManager(c, a.program)
	a.connected = true
	a.focus = focusLocal

	files, err := c.List("/")
	if err != nil {
		a.log = a.log.Add("Error listing remote directory: "+err.Error(), LogError)
		return a, nil
	}

	a.remote = a.remote.WithFiles(files, "/")
	_, panelH, _ := a.heights()
	a.remote = a.remote.SetSize(a.width/2, panelH)
	a.log = a.log.Add("Connecting to "+msg.Host, LogSuccess)

	return a, nil
}

func (a App) handleNavigate(msg NavigateMsg) (App, tea.Cmd) {
	if msg.Panel == "LOCAL" {
		return a, loadLocalDir(msg.Path)
	}

	if !a.connected {
		return a, nil
	}

	files, err := a.client.List(msg.Path)
	if err != nil {
		a.log = a.log.Add("Error: "+err.Error(), LogError)
		return a, nil
	}

	a.remote = a.remote.WithFiles(files, msg.Path)
	_, panelH, _ := a.heights()
	a.remote = a.remote.SetSize(a.width/2, panelH)
	return a, nil
}

func (a App) handleTransfer(msg TransferMsg) (App, tea.Cmd) {
	if !a.connected {
		a.log = a.log.Add("No active connection", LogError)
		return a, nil
	}

	var jobs []transfer.Job

	if msg.SourcePanel == "LOCAL" {
		for _, f := range msg.Files {
			jobs = append(jobs, transfer.Job{
				File:       f,
				LocalPath:  a.local.path,
				RemotePath: a.remote.path,
				Direction:  transfer.Upload,
			})
		}
	} else {
		for _, f := range msg.Files {
			jobs = append(jobs, transfer.Job{
				File:       f,
				LocalPath:  a.local.path,
				RemotePath: a.remote.path,
				Direction:  transfer.Download,
			})
		}
	}

	a.manager.Enqueue(jobs)
	return a, nil
}

func (a App) handleTransferDone(_ TransferDoneMsg) (App, tea.Cmd) {
	var cmds []tea.Cmd
	cmds = append(cmds, loadLocalDir(a.local.path))

	if a.connected {
		remotePath := a.remote.path
		c := a.client
		cmds = append(cmds, func() tea.Msg {
			files, err := c.List(remotePath)
			if err != nil {
				return LogMsg{Message: "Error refreshing remote panel: " + err.Error(), Level: LogError}
			}
			return RemoteDirLoadedMsg{Path: remotePath, Files: files}
		})
	}

	return a, tea.Batch(cmds...)
}

func loadLocalDir(path string) tea.Cmd {
	return func() tea.Msg {
		files, err := listLocalDir(path)
		if err != nil {
			return LogMsg{
				Message: "Error listing local directory: " + err.Error(),
				Level:   LogError,
			}
		}
		return LocalDirLoadedMsg{Path: path, Files: files}
	}
}

type LocalDirLoadedMsg struct {
	Path  string
	Files []model.FileInfo
}

type RemoteDirLoadedMsg struct {
	Path  string
	Files []model.FileInfo
}

type TransferDoneMsg = shared.TransferDoneMsg
