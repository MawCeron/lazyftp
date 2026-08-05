package ui

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
	"github.com/MawCeron/lazyftp/internal/transfer"
	"github.com/charmbracelet/bubbles/spinner"
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

	connected    bool
	connecting   bool
	connectStart time.Time
	spinner      spinner.Model
	verbose      bool
	protoLog     *shared.LineBuffer
}

// The outcome of a connection attempt, which now runs off the update loop.
type connectedMsg struct {
	client client.Client
	addr   string
}

type connectFailedMsg struct {
	err error
}

func NewApp(p func() *tea.Program, verbose bool) App {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}

	app := App{
		focus:     focusConnectionBar,
		connBar:   NewConnectionBar(),
		local:     NewPanel("LOCAL", true),
		remote:    NewPanel("REMOTE", false),
		processes: NewProcessesPanel(),
		log:       NewLogPanel(),
		spinner:   spinner.New(spinner.WithSpinner(spinner.Dot)),
		program:   p,
		verbose:   verbose,
	}
	app.local.path = home
	return app
}

// drainProtoLog moves buffered protocol lines into the log panel. It is called
// once the update loop is free, never from inside a blocking call.
func (a App) drainProtoLog() App {
	if a.protoLog == nil {
		return a
	}
	for _, line := range a.protoLog.Drain() {
		a.log = a.log.Add(line, LogInfo)
	}
	return a
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

	a = a.drainProtoLog()

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

	case spinner.TickMsg:
		// Ticking stops by not asking for the next one.
		if !a.connecting {
			return a, nil
		}
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	case connectedMsg:
		return a.handleConnected(msg)

	case connectFailedMsg:
		return a.handleConnectFailed(msg)

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

	// While an attempt is running the footer says so instead. Movement is the
	// point: a still screen cannot be told apart from a hung one.
	if a.connecting {
		elapsed := time.Since(a.connectStart).Truncate(100 * time.Millisecond)
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(a.width).
			Render(fmt.Sprintf("%s connecting… %s", a.spinner.View(), elapsed))
	}

	var hints []string
	switch a.focus {
	case focusConnectionBar:
		hints = []string{
			hint("Tab", "next field"),
			hint("Shift+Tab", "prev field"),
			hint("←/→", "protocol"),
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
		port = msg.Protocol.DefaultPort()
	}

	if a.client != nil {
		a.client.Disconnect()
	}

	// A typed nil would satisfy the io.Writer interface and be written to.
	var logger io.Writer
	if a.verbose {
		a.protoLog = &shared.LineBuffer{}
		logger = a.protoLog
	}
	c := client.New(msg.Protocol, logger)
	addr := net.JoinHostPort(msg.Host, strconv.Itoa(port))

	a.connecting = true
	a.connectStart = time.Now()
	a.log = a.log.Add("Connecting to "+addr+" over "+msg.Protocol.String(), LogInfo)

	attempt := func() tea.Msg {
		if err := c.Connect(msg.Host, msg.User, msg.Pass, port); err != nil {
			return connectFailedMsg{err: err}
		}
		return connectedMsg{client: c, addr: addr}
	}

	// The spinner is what keeps the update loop turning while the attempt runs,
	// so the elapsed time advances and buffered protocol lines reach the log.
	return a, tea.Batch(attempt, a.spinner.Tick)
}

func (a App) handleConnected(msg connectedMsg) (App, tea.Cmd) {
	a.connecting = false
	a.client = msg.client
	a.manager = transfer.NewManager(msg.client, a.program)
	a.connected = true
	a.focus = focusLocal
	a.log = a.log.Add("Connected to "+msg.addr, LogSuccess)

	return a, loadRemoteDir(msg.client, "/")
}

func (a App) handleConnectFailed(msg connectFailedMsg) (App, tea.Cmd) {
	a.connecting = false
	a.log = a.log.Add("Error connecting: "+msg.err.Error(), LogError)
	return a, nil
}

func (a App) handleNavigate(msg NavigateMsg) (App, tea.Cmd) {
	if msg.Panel == "LOCAL" {
		return a, loadLocalDir(msg.Path)
	}

	if !a.connected {
		return a, nil
	}

	return a, loadRemoteDir(a.client, msg.Path)
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
		cmds = append(cmds, loadRemoteDir(a.client, a.remote.path))
	}

	return a, tea.Batch(cmds...)
}

// loadRemoteDir lists a remote directory off the update loop. Listing is a
// round trip to the server, and doing it inline holds the interface still for
// as long as the server takes to answer.
func loadRemoteDir(c client.Client, path string) tea.Cmd {
	return func() tea.Msg {
		files, err := c.List(path)
		if err != nil {
			return LogMsg{
				Message: "Error listing remote directory: " + err.Error(),
				Level:   LogError,
			}
		}
		return RemoteDirLoadedMsg{Path: path, Files: files}
	}
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
