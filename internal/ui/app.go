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

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
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
		return a, nil

	case RemoteDirLoadedMsg:
		a.remote = a.remote.WithFiles(msg.Files, msg.Path)
		return a, nil

	case TransferDoneMsg:
		return a.handleTransferDone(msg)
	}

	// procesos y log siempre escuchan
	a.processes, _ = a.processes.Update(msg)
	a.log, _ = a.log.Update(msg)

	// delegar al componente enfocado
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

	// alturas: conexión ~3, paneles 55%, bottom 35%, hints 1
	connView := a.connBar.View(a.width, a.focus == focusConnectionBar)
	connHeight := 3

	hintsHeight := 1
	bottomHeight := 8 // mínimo garantizado para processes y log
	panelHeight := a.height - connHeight - bottomHeight - hintsHeight - 2
	if panelHeight < 6 {
		panelHeight = 6
	}
	// recalcular bottom con el espacio real restante
	bottomHeight = a.height - connHeight - panelHeight - hintsHeight - 2
	if bottomHeight < 6 {
		bottomHeight = 6
	}

	panelWidth := a.width / 2
	localView := a.local.View(panelWidth, panelHeight, a.focus == focusLocal)
	remoteView := a.remote.View(panelWidth, panelHeight, a.focus == focusRemote)
	panels := lipgloss.JoinHorizontal(lipgloss.Top, localView, remoteView)

	bottomWidth := a.width / 2
	processesView := a.processes.View(bottomWidth, bottomHeight)
	logView := a.log.View(bottomWidth, bottomHeight)
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, processesView, logView)

	hints := a.hintsView()

	return lipgloss.JoinVertical(lipgloss.Left, connView, panels, bottom, hints)
}

func (a App) hintsView() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238"))

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

	line := strings.Join(hints, sep)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(a.width).
		Render(line)
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
		a.log = a.log.Add("Error conectando: "+err.Error(), LogError)
		return a, nil
	}

	a.client = c
	a.manager = transfer.NewManager(c, a.program)
	a.connected = true
	a.focus = focusLocal

	files, err := c.List("/")
	if err != nil {
		a.log = a.log.Add("Error listando directorio remoto: "+err.Error(), LogError)
		return a, nil
	}

	a.remote = a.remote.WithFiles(files, "/")
	a.log = a.log.Add("Conectado a "+msg.Host, LogSuccess)

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
	return a, nil
}

func (a App) handleTransfer(msg TransferMsg) (App, tea.Cmd) {
	if !a.connected {
		a.log = a.log.Add("No hay conexión activa", LogError)
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

func (a App) handleTransferDone(msg TransferDoneMsg) (App, tea.Cmd) {
	// recargar ambos paneles para reflejar cambios
	var cmds []tea.Cmd

	cmds = append(cmds, loadLocalDir(a.local.path))

	if a.connected {
		remotePath := a.remote.path
		c := a.client
		cmds = append(cmds, func() tea.Msg {
			files, err := c.List(remotePath)
			if err != nil {
				return LogMsg{Message: "Error recargando panel remoto: " + err.Error(), Level: LogError}
			}
			return RemoteDirLoadedMsg{Path: remotePath, Files: files}
		})
	}

	return a, tea.Batch(cmds...)
}

// Cmds y mensajes auxiliares

func loadLocalDir(path string) tea.Cmd {
	return func() tea.Msg {
		files, err := listLocalDir(path)
		if err != nil {
			return LogMsg{
				Message: "Error listando directorio local: " + err.Error(),
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

// TransferDoneMsg se emite desde manager cuando termina una transferencia exitosa
type TransferDoneMsg = shared.TransferDoneMsg
