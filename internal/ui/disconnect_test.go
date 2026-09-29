package ui

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/transfer"
)

var ctrlX = tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl}

type failingClient struct{ stubClient }

func (failingClient) Disconnect() error { return errors.New("broken pipe") }

// blockingClient holds every upload until release is closed.
type blockingClient struct {
	stubClient
	release chan struct{}
}

func (b *blockingClient) Upload(local, remote string, p func(int64)) error {
	<-b.release
	return nil
}

type idleModel struct{}

func (idleModel) Init() tea.Cmd                       { return nil }
func (idleModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return idleModel{}, nil }
func (idleModel) View() tea.View                      { return tea.NewView("") }

func headlessProgram(t *testing.T) *tea.Program {
	t.Helper()
	p := tea.NewProgram(idleModel{}, tea.WithInput(strings.NewReader("")), tea.WithOutput(io.Discard),
		tea.WithoutRenderer(), tea.WithoutSignalHandler(), tea.WithoutSignals())
	go p.Run()
	t.Cleanup(func() { p.Quit(); p.Wait() })
	return p
}

func connectedTo(c client.Client, m *transfer.Manager) App {
	a := NewApp(nil, false, nil, "dev", false)
	a.client, a.manager, a.connected = c, m, true
	a.connUser, a.connAddr, a.connProtocol = "ana", "nas.lan:21", client.FTP
	return a
}

func lastLog(a App) string { return a.log.entries[len(a.log.entries)-1].Message }

func TestCtrlXDisconnectsAndLeavesTheAppOffline(t *testing.T) {
	stub := &stubClient{}
	a := connectedTo(stub, nil)
	a.focus = focusRemote
	a.remote, _ = a.remote.WithFiles([]model.FileInfo{{Name: "a.txt"}}, "/home/ana")

	got, _ := a.Update(ctrlX)
	a = got.(App)

	if !stub.disconnected || a.connected || a.client != nil || a.manager != nil {
		t.Fatalf("still connected: disconnected=%v connected=%v", stub.disconnected, a.connected)
	}
	if a.focus != focusLocal || len(a.remote.files) != 0 || a.remote.path != "/" {
		t.Errorf("stale remote: focus=%v files=%d path=%q", a.focus, len(a.remote.files), a.remote.path)
	}
	if got := lastLog(a); got != "Disconnected from nas.lan:21" {
		t.Errorf("log %q", got)
	}
	if !strings.Contains(a.statusLine(), "OFFLINE") {
		t.Errorf("status line %q", a.statusLine())
	}
}

func TestADisconnectErrorIsLoggedNotDiscarded(t *testing.T) {
	got, _ := connectedTo(&failingClient{}, nil).Update(ctrlX)
	if got := lastLog(got.(App)); !strings.Contains(got, "broken pipe") || !strings.Contains(got, "nas.lan:21") {
		t.Errorf("log %q", got)
	}
}

func TestConnectingElsewhereLogsTheOldConnectionClosing(t *testing.T) {
	stub := &stubClient{}
	a := connectedTo(stub, nil)
	a.log = a.log.Add("marker", LogInfo)

	a, _ = a.handleConnect(ConnectMsg{Protocol: client.FTP, Host: "other.lan", Port: "21"})
	msgs := ""
	for _, e := range a.log.entries {
		msgs += e.Message + "\n"
	}
	if !stub.disconnected || !strings.Contains(msgs, "Disconnected from nas.lan:21") || a.connected {
		t.Errorf("disconnected=%v connected=%v log:\n%s", stub.disconnected, a.connected, msgs)
	}

	a, _ = connectedTo(&failingClient{}, nil).handleConnect(ConnectMsg{Protocol: client.FTP, Host: "other.lan", Port: "21"})
	msgs = ""
	for _, e := range a.log.entries {
		msgs += e.Message + "\n"
	}
	if !strings.Contains(msgs, "broken pipe") {
		t.Errorf("a failing Disconnect vanished:\n%s", msgs)
	}
}

func TestATransferInFlightBlocksDisconnectingAndReconnecting(t *testing.T) {
	blocker := &blockingClient{release: make(chan struct{})}
	m := transfer.NewManager(blocker, func() *tea.Program { return headlessProgram(t) })
	m.Enqueue([]transfer.Job{{File: model.FileInfo{Name: "big.bin"}, LocalPath: t.TempDir(), RemotePath: "/", Direction: transfer.Upload}})
	t.Cleanup(func() { close(blocker.release) })

	a := connectedTo(blocker, m)
	got, _ := a.Update(ctrlX)
	a = got.(App)
	if !a.connected || blocker.disconnected {
		t.Fatal("disconnected with a transfer still running")
	}
	if !strings.Contains(lastLog(a), "1 transfer") {
		t.Errorf("log %q", lastLog(a))
	}

	a, cmd := a.handleConnect(ConnectMsg{Protocol: client.FTP, Host: "other.lan", Port: "21"})
	if !a.connected || blocker.disconnected || cmd != nil {
		t.Fatal("reconnected over a running transfer")
	}
}

func TestDisconnectWaitsForNothingOnceTransfersFinish(t *testing.T) {
	blocker := &blockingClient{release: make(chan struct{})}
	m := transfer.NewManager(blocker, func() *tea.Program { return headlessProgram(t) })
	m.Enqueue([]transfer.Job{{File: model.FileInfo{Name: "big.bin"}, LocalPath: t.TempDir(), RemotePath: "/", Direction: transfer.Upload}})
	close(blocker.release)
	for i := 0; m.Active() > 0; i++ {
		if i > 200 {
			t.Fatal("transfer never finished")
		}
		time.Sleep(10 * time.Millisecond)
	}
	got, _ := connectedTo(blocker, m).Update(ctrlX)
	if got.(App).connected {
		t.Fatal("still connected after the transfer finished")
	}
}

func TestADirectoryListingArrivingAfterDisconnectIsDropped(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	got, _ := a.Update(RemoteDirLoadedMsg{Files: []model.FileInfo{{Name: "ghost"}}, Path: "/"})
	if n := len(got.(App).remote.files); n != 0 {
		t.Fatalf("offline app grew %d remote files", n)
	}
}

func TestCtrlXDoesNothingWhileOffline(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	before := len(a.log.entries)
	got, _ := a.Update(ctrlX)
	if len(got.(App).log.entries) != before {
		t.Fatal("logged a disconnect that never happened")
	}
}

func TestStatusLineHintsTheDisconnectKeyOnlyWhenItFits(t *testing.T) {
	a := connectedTo(&stubClient{}, nil)
	a.width = 100
	if !strings.Contains(a.statusLine(), "Ctrl+X") {
		t.Error("no hint on a wide terminal")
	}
	a.width = 40
	if strings.Contains(a.statusLine(), "Ctrl+X") {
		t.Error("hint crowded a narrow status line")
	}
}

func TestAReopenedSessionIsLoggedAndNothingElseChanges(t *testing.T) {
	stub := &stubClient{}
	a := connectedTo(stub, nil)
	a.focus = focusRemote
	got, _ := a.Update(sessionMsg{client: stub})
	a = got.(App)
	if !a.connected || a.focus != focusRemote || !strings.Contains(lastLog(a), "reopened") {
		t.Fatalf("connected=%v focus=%v log=%q", a.connected, a.focus, lastLog(a))
	}
}

func TestASessionThatCannotBeReopenedGoesOfflineWithTheDialogReady(t *testing.T) {
	stub := &stubClient{}
	a := connectedTo(stub, nil)
	a.connBar = fillField(a.connBar, fieldHost, "nas.lan")
	a.focus = focusRemote

	got, _ := a.Update(sessionMsg{client: stub, err: errors.New("connection refused")})
	a = got.(App)
	if a.connected || a.focus != focusConnectionBar || a.connBar.inputs[fieldHost].Value() != "nas.lan" {
		t.Fatalf("connected=%v focus=%v host=%q", a.connected, a.focus, a.connBar.inputs[fieldHost].Value())
	}
	if l := lastLog(a); !strings.Contains(l, "connection refused") || !strings.Contains(l, "Enter") {
		t.Errorf("log %q", l)
	}
}

func TestAReportFromAnOldConnectionIsIgnored(t *testing.T) {
	old, current := &stubClient{}, &stubClient{}
	a := connectedTo(current, nil)
	got, _ := a.Update(sessionMsg{client: old, err: errors.New("late")})
	if !got.(App).connected {
		t.Fatal("a dead connection the user already left took the live one down")
	}
}
