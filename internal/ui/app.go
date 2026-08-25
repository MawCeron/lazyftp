package ui

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

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
	focusLog
	focusProcesses
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

	version       string
	highlightDiff bool

	themeResolved bool
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

func NewApp(p func() *tea.Program, verbose bool, logFile io.Writer, version string, highlightDiff bool) App {
	app := App{
		focus:         focusConnectionBar,
		connBar:       NewConnectionBar(),
		local:         NewPanel("Local", true),
		remote:        NewPanel("Remote", false),
		processes:     NewProcessesPanel(),
		log:           NewLogPanel(logFile),
		spinner:       spinner.New(spinner.WithSpinner(spinner.Dot)),
		program:       p,
		verbose:       verbose,
		version:       version,
		highlightDiff: highlightDiff,
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
	return tea.Batch(loadLocalDir(a.local.path), tea.RequestBackgroundColor, themeFallbackTimeout())
}

// themeFallbackTimeout guards tea.RequestBackgroundColor: bubbletea sends the
// OSC 11 query fire-and-forget, with no timeout of its own, so a terminal or
// multiplexer that never answers (common enough over tmux/SSH) would
// otherwise leave the theme silently stuck on its dark static default
// forever. If BackgroundColorMsg hasn't resolved it within this window, fall
// back to dark explicitly instead of relying on that coincidence.
const themeFallbackDelay = 300 * time.Millisecond

type themeFallbackMsg struct{}

func themeFallbackTimeout() tea.Cmd {
	return tea.Tick(themeFallbackDelay, func(time.Time) tea.Msg {
		return themeFallbackMsg{}
	})
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

// focusedPanelJumping reports whether the currently focused file panel has
// its jump-to-path input open.
func (a App) focusedPanelJumping() bool {
	switch a.focus {
	case focusLocal:
		return a.local.jumping
	case focusRemote:
		return a.remote.jumping
	default:
		return false
	}
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

// bottomPanelHeight is how tall Processes and Log each render: split when
// stacked in narrow mode, shared in full when side by side. LogPanel needs
// this figure kept in sync via SetSize -- unlike Processes, it persists a
// viewport whose height math (AtBottom, in particular) goes stale otherwise.
func (a App) bottomPanelHeight(bottomH int) int {
	if !a.narrow() {
		return bottomH
	}
	h := bottomH / 2
	if h < 4 {
		h = 4
	}
	return h
}

// focusedPanelFiltering reports whether the currently focused file panel is
// actively capturing filter query input. Global key handling in Update must
// not intercept keys while this is true, or typing a filter query
// containing e.g. "q" or "U" would trigger that action instead of being
// entered as filter text.
func (a App) focusedPanelFiltering() bool {
	switch a.focus {
	case focusLocal:
		return a.local.Filtering()
	case focusRemote:
		return a.remote.Filtering()
	}
	return false
}

// focusedPanelHasFilter reports whether the currently focused file panel has
// a filter active at all -- typing or already applied. Esc must be allowed
// to reach the panel in both states so the list's own keymap can cancel or
// clear it -- see acceptance criteria on #31.
func (a App) focusedPanelHasFilter() bool {
	switch a.focus {
	case focusLocal:
		return a.local.HasFilter()
	case focusRemote:
		return a.remote.HasFilter()
	}
	return false
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	a = a.drainProtoLog()

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		_, panelH, bottomH := a.heights()
		panelW := a.panelWidth()
		a.local = a.local.SetSize(panelW, panelH)
		a.remote = a.remote.SetSize(panelW, panelH)
		a.log = a.log.SetSize(panelW, a.bottomPanelHeight(bottomH))
		a.processes = a.processes.SetSize(panelW, a.bottomPanelHeight(bottomH))
		return a, nil

	case tea.BackgroundColorMsg:
		a.themeResolved = true
		SetTheme(msg.IsDark())
		return a, nil

	case themeFallbackMsg:
		if !a.themeResolved {
			a.themeResolved = true
			SetTheme(true)
		}
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

		// A panel's own jump-to-path input, and a filter query being typed,
		// are both modal in the same way: while either is focused, global
		// bindings must not steal keystrokes that input would otherwise
		// receive (a bare "q" is a perfectly normal path character, and just
		// as valid in a filter query).
		jumping := a.focusedPanelJumping()
		filtering := a.focusedPanelFiltering()

		if a.focus != focusConnectionBar && !jumping && !filtering {
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
		// Tab stays within whichever group has focus -- Local/Remote or
		// Log/Processes; Shift+Tab is what moves between the two groups.
		// Tab alone used to be reversible on its own when there were only
		// two panels total; once Log joined the cycle that stopped being
		// true, which is what this group split restores.
		case key.Matches(msg, keySwitch):
			// Guarded the same way as the global block above: while the
			// focused panel is capturing filter input (or jumping), Tab must
			// reach the list rather than switch focus, or the panel left
			// behind stays stuck in an unfinished filter/jump that swallows
			// its other keys until someone tabs back to finish or cancel it.
			if a.focus != focusConnectionBar && !jumping && !filtering {
				switch a.focus {
				case focusLocal:
					a.focus = focusRemote
				case focusRemote:
					a.focus = focusLocal
				case focusLog:
					a.focus = focusProcesses
				case focusProcesses:
					a.focus = focusLog
				}
				return a, nil
			}
		case key.Matches(msg, keySwitchZone):
			if a.focus != focusConnectionBar && !jumping && !filtering {
				switch a.focus {
				case focusLocal, focusRemote:
					a.focus = focusProcesses
				default:
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
				return a, nil
			}
			if !jumping && !a.focusedPanelHasFilter() {
				return a, nil
			}
			// jumping, or a filter is typing or applied: fall through so the
			// panel's own Update closes/cancels/clears whichever it is.
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
		var filterCmd tea.Cmd
		a.local, filterCmd = a.local.WithFiles(msg.Files, msg.Path)
		_, panelH, _ := a.heights()
		a.local = a.local.SetSize(a.panelWidth(), panelH)
		return a, filterCmd

	case RemoteDirLoadedMsg:
		var filterCmd tea.Cmd
		a.remote, filterCmd = a.remote.WithFiles(msg.Files, msg.Path)
		_, panelH, _ := a.heights()
		a.remote = a.remote.SetSize(a.panelWidth(), panelH)
		return a, filterCmd

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
	case focusLog:
		a.log, cmd = a.log.UpdateFocused(msg)
	case focusProcesses:
		a.processes, cmd = a.processes.UpdateFocused(msg)
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

	// Both fields nil unless --highlight-diff is on: Panel.View treats that
	// as "the flag is off" and drops the indicator column entirely rather
	// than reserving a permanently blank one nobody asked for.
	var localDiff, remoteDiff diffMarks
	if a.highlightDiff {
		localDiff = diffMarks{
			uniqueOnly:  uniqueNames(a.local.files, a.remote.files),
			sizeDiffers: sizeDiffers(a.local.files, a.remote.files),
		}
		remoteDiff = diffMarks{
			uniqueOnly:  uniqueNames(a.remote.files, a.local.files),
			sizeDiffers: sizeDiffers(a.remote.files, a.local.files),
		}
	}

	var panels string
	if a.narrow() {
		// Only the focused file panel renders, full width, switched with
		// the same Tab that already cycles focus between Local and Remote.
		if a.focus == focusRemote {
			panels = a.remote.View(panelW, panelH, true, remoteDiff)
		} else {
			panels = a.local.View(panelW, panelH, a.focus == focusLocal, localDiff)
		}
	} else {
		localView := a.local.View(panelW, panelH, a.focus == focusLocal, localDiff)
		remoteView := a.remote.View(panelW, panelH, a.focus == focusRemote, remoteDiff)
		panels = lipgloss.JoinHorizontal(lipgloss.Top, localView, remoteView)
	}

	logActive := a.focus == focusLog
	processesActive := a.focus == focusProcesses

	var bottom string
	if a.narrow() {
		// Side by side, Processes and Log would each get under 30 columns:
		// stack them instead, splitting the shared height budget.
		stackedH := a.bottomPanelHeight(bottomH)
		processesView := a.processes.View(panelW, stackedH, processesActive)
		logView := a.log.View(panelW, stackedH, logActive)
		bottom = lipgloss.JoinVertical(lipgloss.Left, processesView, logView)
	} else {
		processesView := a.processes.View(panelW, bottomH, processesActive)
		logView := a.log.View(panelW, bottomH, logActive)
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
// the connection state, detail text centered in the plain space next to it,
// and a pill on the far right for the marked-file count -- visible nowhere
// else short of opening Processes or counting checkmarks by eye -- whenever
// there is one.
func (a App) statusLine() string {
	bar := lipgloss.NewStyle().Background(colorBarBg)
	detailStyle := bar.Foreground(colorMuted)

	var leftPill, detail string
	switch {
	case a.connecting:
		elapsed := time.Since(a.connectStart).Truncate(100 * time.Millisecond)
		leftPill = pill(colorAccent, a.spinner.View()+" CONNECTING")
		detail = detailStyle.Render(elapsed.String())

	case a.connected:
		who := fmt.Sprintf("%s@%s", a.connUser, a.connAddr)
		leftPill = pill(colorSuccess, a.connProtocol.String())
		detail = detailStyle.Foreground(colorPrimary).Render(who)

	default:
		leftPill = pill(colorMuted, "OFFLINE")
		detail = detailStyle.Render("Ctrl+L to connect")
	}

	var rightPill string
	if marked := len(a.local.markedFiles()) + len(a.remote.markedFiles()); marked > 0 {
		rightPill = pill(colorMarked, fmt.Sprintf("%d MARKED", marked))
	}

	plainWidth := a.width - lipgloss.Width(leftPill) - lipgloss.Width(rightPill)
	leftPad, rightPad := centerPadding(plainWidth, lipgloss.Width(detail))
	plain := bar.Render(strings.Repeat(" ", leftPad)) + detail + bar.Render(strings.Repeat(" ", rightPad))

	return leftPill + plain + rightPill
}

// centerPadding splits the slack in a span of the given width around
// content contentWidth cells wide, clamped to zero if the content doesn't
// fit at all.
func centerPadding(width, contentWidth int) (left, right int) {
	slack := width - contentWidth
	if slack < 0 {
		return 0, 0
	}
	left = slack / 2
	return left, slack - left
}

// hintsView anchors the app's identity at the left edge, then fills the rest
// of the bar with the same bindings keys.go declares for the help screen,
// trimmed to what's actionable from here -- see footerKeyMap.ShortHelp.
// Leading with the pills means whatever renderHints doesn't use of its width
// budget (it drops a whole hint rather than show one partially) falls off
// the right edge as ordinary trailing space, not a gap stranded between two
// fixed pieces of content.
func (a App) hintsView() string {
	identity := pill(colorAccent, "lazyftp") + pill(colorMuted, a.version)
	gap := lipgloss.NewStyle().Background(colorBarBg).Render("  ")
	leadWidth := lipgloss.Width(identity) + lipgloss.Width(gap)

	km := footerKeyMap{focus: a.focus, connecting: a.connecting, helpOpen: a.helpOpen, jumping: a.focusedPanelJumping()}
	hints := renderHints(km.ShortHelp(), a.width-leadWidth)

	return composeBar(colorBarBg, a.width, identity+gap+hints, "")
}

// renderHints lays out bindings as "key desc" pairs, hand-composed instead
// of through bubbles/help's ShortHelpView: that helper joins a rendered key
// and a rendered desc with a literal, unstyled space, which showed the
// terminal's own background right through the middle of every hint on a
// colored bar. Drops whichever trailing bindings don't fit width, with an
// ellipsis if there's room for one.
func renderHints(bindings []key.Binding, width int) string {
	bar := lipgloss.NewStyle().Background(colorBarBg)
	keyStyle := bar.Bold(true).Foreground(colorEmphasis)
	descStyle := bar.Foreground(colorMuted)
	sep := bar.Foreground(colorMuted).Render(" • ")
	sepWidth := lipgloss.Width(sep)

	var b strings.Builder
	used := 0
	for i, kb := range bindings {
		h := kb.Help()
		item := keyStyle.Render(h.Key) + bar.Render(" ") + descStyle.Render(h.Desc)
		itemWidth := lipgloss.Width(item)

		need := itemWidth
		if i > 0 {
			need += sepWidth
		}
		if used+need > width {
			if used+sepWidth+1 <= width {
				if i > 0 {
					b.WriteString(sep)
				}
				b.WriteString(descStyle.Render("…"))
			}
			break
		}

		if i > 0 {
			b.WriteString(sep)
			used += sepWidth
		}
		b.WriteString(item)
		used += itemWidth
	}
	return b.String()
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
