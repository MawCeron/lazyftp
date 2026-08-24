package ui

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
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
	width    int
	height   int
	focus    focus
	helpOpen bool

	client  client.Client
	manager *transfer.Manager
	program func() *tea.Program

	connBar   ConnectionBar
	local     Panel
	remote    Panel
	processes ProcessesPanel
	log       LogPanel

	connected    bool
	connUser     string
	connAddr     string
	connProtocol client.Protocol

	connecting   bool
	connectSeq   int
	connectStart time.Time
	spinner      spinner.Model
	verbose      bool
	protoLog     *shared.LineBuffer

	version string
}

// seq identifies the attempt, so an abandoned one's result can be dropped.
type connectedMsg struct {
	seq      int
	client   client.Client
	addr     string
	user     string
	protocol client.Protocol
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

func NewApp(p func() *tea.Program, verbose bool, logFile io.Writer, version string) App {
	app := App{
		focus:     focusConnectionBar,
		connBar:   NewConnectionBar(),
		local:     NewPanel("Local", true),
		remote:    NewPanel("Remote", false),
		processes: NewProcessesPanel(),
		log:       NewLogPanel(logFile),
		spinner:   spinner.New(spinner.WithSpinner(spinner.Dot)),
		program:   p,
		verbose:   verbose,
		version:   version,
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

const (
	minWidth  = 60 // below this (or minHeight), show a "too small" message instead
	minHeight = 20

	standardBreakpoint = 80 // below this: single file panel, Tab-switched
)

// tooSmall guards the floor: below it nothing else is safe to lay out.
func (a App) tooSmall() bool {
	return a.width < minWidth || a.height < minHeight
}

// narrow is true below the standard 80-column floor, where two file panels
// side by side would be too cramped to be useful. Below it, only the
// focused panel renders, full width, switched with the same Tab that
// already cycles focus.
func (a App) narrow() bool {
	return a.width < standardBreakpoint
}

func (a App) panelWidth() int {
	if a.narrow() {
		return a.width
	}
	return a.width / 2
}

func (a App) heights() (statusH, panelH, bottomH int) {
	statusH = 1  // connection status line
	bottomH = 10 // Processes + Log minimal fixed
	hintsH := 1
	panelH = a.height - statusH - bottomH - hintsH
	if panelH < 8 {
		panelH = 8
	}
	bottomH = a.height - statusH - panelH - hintsH
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
		panelW := a.panelWidth()
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

		// The help screen is modal: while it's open, every key either closes
		// it or is swallowed, same as the connection dialog owning the
		// keyboard while it has focus.
		if a.helpOpen {
			if key.Matches(msg, keyEsc) || key.Matches(msg, keyHelp) {
				a.helpOpen = false
			}
			return a, nil
		}

		if a.focus != focusConnectionBar {
			switch {
			case key.Matches(msg, keyQuit):
				if a.client != nil {
					a.client.Disconnect()
				}
				return a, tea.Quit

			case key.Matches(msg, keyHelp):
				a.helpOpen = true
				return a, nil

			case key.Matches(msg, keyUpload):
				return a.handleDirectTransfer("Local", a.local.markedFiles())

			case key.Matches(msg, keyDownload):
				return a.handleDirectTransfer("Remote", a.remote.markedFiles())
			}
		}

		switch {
		case key.Matches(msg, keyConnect):
			a.focus = focusConnectionBar
			return a, nil
		case key.Matches(msg, keySwitch):
			if a.focus != focusConnectionBar {
				if a.focus == focusLocal {
					a.focus = focusRemote
				} else {
					a.focus = focusLocal
				}
				return a, nil
			}
		case key.Matches(msg, keyEsc):
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
		a.local = a.local.SetSize(a.panelWidth(), panelH)
		return a, nil

	case RemoteDirLoadedMsg:
		a.remote = a.remote.WithFiles(msg.Files, msg.Path)
		_, panelH, _ := a.heights()
		a.remote = a.remote.SetSize(a.panelWidth(), panelH)
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
	if a.tooSmall() {
		return a.tooSmallView()
	}

	_, panelH, bottomH := a.heights()
	panelW := a.panelWidth()

	status := a.statusLine()
	hints := a.hintsView()

	// The connection dialog and the help screen are both fixed-size overlays,
	// not a mode any panel needs to stay visible under: blanking them avoids
	// the overlay cutting their text off mid-word wherever it overlaps.
	if a.focus == focusConnectionBar || a.helpOpen {
		base := lipgloss.JoinVertical(lipgloss.Left, status, blankArea(a.width, panelH), blankArea(a.width, bottomH), hints)
		if a.helpOpen {
			// Capped to the blank canvas itself (panelH+bottomH), not the
			// full height: the status line above it and the hints below
			// are not part of that canvas and must stay clear.
			return a.withOverlay(base, helpScreenView(a.width, panelH+bottomH))
		}
		return a.withOverlay(base, a.connBar.View(a.width))
	}

	var panels string
	if a.narrow() {
		// Only the focused file panel renders, full width, switched with
		// the same Tab that already cycles focus between Local and Remote.
		if a.focus == focusRemote {
			panels = a.remote.View(panelW, panelH, true)
		} else {
			panels = a.local.View(panelW, panelH, a.focus == focusLocal)
		}
	} else {
		localView := a.local.View(panelW, panelH, a.focus == focusLocal)
		remoteView := a.remote.View(panelW, panelH, a.focus == focusRemote)
		panels = lipgloss.JoinHorizontal(lipgloss.Top, localView, remoteView)
	}

	var bottom string
	if a.narrow() {
		// Side by side, Processes and Log would each get under 30 columns:
		// stack them instead, splitting the shared height budget.
		stackedH := bottomH / 2
		if stackedH < 4 {
			stackedH = 4
		}
		processesView := a.processes.View(panelW, stackedH)
		logView := a.log.View(panelW, stackedH)
		bottom = lipgloss.JoinVertical(lipgloss.Left, processesView, logView)
	} else {
		processesView := a.processes.View(panelW, bottomH)
		logView := a.log.View(panelW, bottomH)
		bottom = lipgloss.JoinHorizontal(lipgloss.Top, processesView, logView)
	}

	return lipgloss.JoinVertical(lipgloss.Left, status, panels, bottom, hints)
}

// blankArea returns height blank lines, each width cells wide, so the
// connection overlay has an empty canvas to sit on instead of the panels.
func blankArea(width, height int) string {
	if height <= 0 {
		return ""
	}
	line := strings.Repeat(" ", width)
	lines := make([]string, height)
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func (a App) tooSmallView() string {
	msg := fmt.Sprintf("terminal too small — need at least %dx%d", minWidth, minHeight)
	return lipgloss.NewStyle().
		Width(a.width).
		Height(a.height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(colorMuted).
		Render(msg)
}

// withOverlay floats content centered over base using lipgloss's layer
// compositor.
func (a App) withOverlay(base, overlay string) string {
	ow, oh := lipgloss.Width(overlay), lipgloss.Height(overlay)

	x, y := (a.width-ow)/2, (a.height-oh)/2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	bg := lipgloss.NewLayer(base)
	fg := lipgloss.NewLayer(overlay).X(x).Y(y).Z(1)
	return lipgloss.NewCompositor(bg, fg).Render()
}

// statusLine is a segmented bar, lualine/airline-style: a colored pill for
// the connection state, plain detail text next to it, and a pill on the far
// right for the marked-file count -- visible nowhere else short of opening
// Processes or counting checkmarks by eye -- whenever there is one.
func (a App) statusLine() string {
	bar := lipgloss.NewStyle().Background(colorBarBg)
	detail := bar.Foreground(colorMuted)

	var left string
	switch {
	case a.connecting:
		elapsed := time.Since(a.connectStart).Truncate(100 * time.Millisecond)
		left = pill(colorAccent, a.spinner.View()+" CONNECTING") +
			bar.Render(" ") + detail.Render(elapsed.String())

	case a.connected:
		who := fmt.Sprintf("%s@%s", a.connUser, a.connAddr)
		left = pill(colorSuccess, a.connProtocol.String()) +
			bar.Render(" ") + detail.Foreground(colorPrimary).Render(who)

	default:
		left = pill(colorMuted, "OFFLINE") +
			bar.Render(" ") + detail.Render("Ctrl+L to connect")
	}

	var right string
	if marked := len(a.local.markedFiles()) + len(a.remote.markedFiles()); marked > 0 {
		right = pill(colorMarked, fmt.Sprintf("%d MARKED", marked))
	}

	return composeBar(colorBarBg, a.width, left, right)
}

// hintsView renders the same bindings keys.go declares for the help screen,
// trimmed to what's actionable from here -- see footerKeyMap.ShortHelp. Two
// pills anchor the right edge with the app's identity; unlike the per-key
// hints, that doesn't change with focus, so it isn't worth its own pass
// through footerKeyMap. SetWidth gives the hints only the width left after
// the pills take their share, so they truncate gracefully with an ellipsis
// instead of colliding with the pills or overflowing.
func (a App) hintsView() string {
	bar := lipgloss.NewStyle().Background(colorBarBg)

	right := pill(colorAccent, "lazyftp") + pill(colorMuted, a.version)
	rightWidth := lipgloss.Width(right)

	hm := help.New()
	hm.Styles.ShortKey = bar.Bold(true).Foreground(colorEmphasis)
	hm.Styles.ShortDesc = bar.Foreground(colorMuted)
	hm.Styles.ShortSeparator = bar.Foreground(colorMuted)
	hm.Styles.Ellipsis = bar.Foreground(colorMuted)
	hm.SetWidth(a.width - rightWidth)

	km := footerKeyMap{focus: a.focus, connecting: a.connecting, helpOpen: a.helpOpen}
	return composeBar(colorBarBg, a.width, hm.View(km), right)
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
		return connectedMsg{seq: seq, client: c, addr: addr, user: msg.User, protocol: msg.Protocol}
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
	a.connUser = msg.user
	a.connAddr = msg.addr
	a.connProtocol = msg.protocol
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
	if msg.Panel == "Local" {
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

	if msg.SourcePanel == "Local" {
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
