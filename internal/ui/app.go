package ui

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
	"github.com/MawCeron/lazyftp/internal/transfer"
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
	connectSeq   int
	connectStart time.Time
	spinner      spinner.Model
	verbose      bool
	protoLog     *shared.LineBuffer
}

// seq identifies the attempt, so an abandoned one's result can be dropped.
type connectedMsg struct {
	seq    int
	client client.Client
	addr   string
}

type connectFailedMsg struct {
	seq int
	err error
}

func startDir() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "/"
}

func NewApp(p func() *tea.Program, verbose bool, logFile io.Writer) App {
	app := App{
		focus:     focusConnectionBar,
		connBar:   NewConnectionBar(),
		local:     NewPanel("LOCAL", true),
		remote:    NewPanel("REMOTE", false),
		processes: NewProcessesPanel(),
		log:       NewLogPanel(logFile),
		spinner:   spinner.New(spinner.WithSpinner(spinner.Dot)),
		program:   p,
		verbose:   verbose,
	}
	app.local.path = startDir()
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
	return tea.Batch(loadLocalDir(a.local.path), tea.RequestBackgroundColor)
}

func (a App) heights() (connH, panelH, bottomH int) {
	connH = 5    // ConnectionBar field
	bottomH = 10 // Processes + Log minimal fixed
	hintsH := 1
	panelH = a.height - connH - bottomH - hintsH
	if panelH < 8 {
		panelH = 8
	}
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

	case tea.BackgroundColorMsg:
		SetTheme(msg.IsDark())
		return a, nil

	case tea.KeyPressMsg:
		// Ctrl+C must always quit cleanly, from every focus state -- unlike
		// "q", it's a control chord that a text field would never receive as
		// literal input.
		if msg.String() == "ctrl+c" {
			if a.client != nil {
				a.client.Disconnect()
			}
			return a, tea.Quit
		}

		if a.focus != focusConnectionBar {
			switch msg.String() {
			case "q", "Q":
				if a.client != nil {
					a.client.Disconnect()
				}
				return a, tea.Quit

			case "U":
				return a.handleDirectTransfer("LOCAL", a.local.markedFiles())

			case "D":
				return a.handleDirectTransfer("REMOTE", a.remote.markedFiles())
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
			if a.connecting {
				// Neither client takes a context, so the attempt is let go of
				// rather than cancelled: it ends on its own timeout.
				a.connectSeq++
				a.connecting = false
				a.log = a.log.Add("Connection attempt abandoned", LogInfo)
				return a, nil
			}
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

func (a App) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	return v
}

func (a App) render() string {
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
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(colorEmphasis)
	descStyle := lipgloss.NewStyle().Foreground(colorMuted)
	sepStyle := lipgloss.NewStyle().Foreground(colorMuted)
	sep := sepStyle.Render(" | ")

	hint := func(key, desc string) string {
		return keyStyle.Render(key) + ": " + descStyle.Render(desc)
	}

	if a.connecting {
		elapsed := time.Since(a.connectStart).Truncate(100 * time.Millisecond)
		return lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(a.width).
			Render(fmt.Sprintf("%s connecting… %s   %s", a.spinner.View(), elapsed,
				hint("Esc", "abandon")))
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
			hint("U/D", "upload/download marked"),
			hint("Tab", "switch panel"),
			hint("Ctrl+L", "connection"),
			hint("q", "quit"),
		}
	}

	return lipgloss.NewStyle().
		Foreground(colorMuted).
		Width(a.width).
		Render(strings.Join(hints, sep))
}

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
	a.connectSeq++
	seq := a.connectSeq
	a.log = a.log.Add("Connecting to "+addr+" over "+msg.Protocol.String(), LogInfo)

	attempt := func() tea.Msg {
		if err := c.Connect(msg.Host, msg.User, msg.Pass, port); err != nil {
			return connectFailedMsg{seq: seq, err: err}
		}
		return connectedMsg{seq: seq, client: c, addr: addr}
	}

	// The spinner keeps the update loop turning, which advances the elapsed time
	// and drains buffered protocol lines into the log.
	return a, tea.Batch(attempt, a.spinner.Tick)
}

func (a App) handleConnected(msg connectedMsg) (App, tea.Cmd) {
	if msg.seq != a.connectSeq {
		// Abandoned, but it connected anyway. Close what nobody holds.
		msg.client.Disconnect()
		return a, nil
	}

	a.connecting = false
	a.client = msg.client
	a.manager = transfer.NewManager(msg.client, a.program)
	a.connected = true
	a.focus = focusLocal
	a.log = a.log.Add("Connected to "+msg.addr, LogSuccess)

	return a, loadRemoteDir(msg.client, "/")
}

func (a App) handleConnectFailed(msg connectFailedMsg) (App, tea.Cmd) {
	if msg.seq != a.connectSeq {
		return a, nil
	}

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

// handleDirectTransfer is U/D: transfer marked files in a given direction
// regardless of which panel currently has focus.
func (a App) handleDirectTransfer(sourcePanel string, files []model.FileInfo) (App, tea.Cmd) {
	if len(files) == 0 {
		a.log = a.log.Add("No files marked in "+sourcePanel, LogError)
		return a, nil
	}
	return a.handleTransfer(TransferMsg{SourcePanel: sourcePanel, Files: files})
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
