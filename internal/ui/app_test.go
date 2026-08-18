package ui

import (
	"errors"
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/model"
)

// stubClient stands in for a server. Only Disconnect is observed: abandoning an
// attempt has to close whatever the attempt opened.
type stubClient struct {
	disconnected bool
}

func (s *stubClient) Connect(host, user, pass string, port int) error { return nil }
func (s *stubClient) Disconnect() error                               { s.disconnected = true; return nil }
func (s *stubClient) List(path string) ([]model.FileInfo, error)      { return nil, nil }
func (s *stubClient) Mkdir(path string) error                         { return nil }
func (s *stubClient) Upload(local, remote string, p func(int64)) error {
	return nil
}
func (s *stubClient) Download(remote, local string, p func(int64)) error {
	return nil
}

var _ client.Client = (*stubClient)(nil)

func connecting() App {
	a := NewApp(nil, false, nil)
	a.connecting = true
	a.connectSeq = 1
	return a
}

func TestEscAbandonsAnAttemptInProgress(t *testing.T) {
	a := connecting()

	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	a = model.(App)

	if a.connecting {
		t.Error("the attempt is still marked as running after Esc")
	}
	if a.connectSeq == 1 {
		t.Error("the attempt was not invalidated, so its result would still be taken")
	}
}

// The attempt keeps running after Esc, because neither client library accepts a
// context. Its result must not connect the application behind the user's back.
func TestAnAbandonedAttemptDoesNotConnect(t *testing.T) {
	a := connecting()

	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	a = model.(App)

	stub := &stubClient{}
	model, _ = a.Update(connectedMsg{seq: 1, client: stub, addr: "example.org:21"})
	a = model.(App)

	if a.connected {
		t.Error("an abandoned attempt connected the application")
	}
	if a.client != nil {
		t.Error("an abandoned attempt installed its client")
	}
	if !stub.disconnected {
		t.Error("the abandoned attempt left its connection open")
	}
}

func TestAnAbandonedAttemptDoesNotReportItsError(t *testing.T) {
	a := connecting()

	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	a = model.(App)
	before := len(a.log.entries)

	model, _ = a.Update(connectFailedMsg{seq: 1, err: errors.New("i/o timeout")})
	a = model.(App)

	if len(a.log.entries) != before {
		t.Errorf("an abandoned attempt logged its failure: %q", a.log.entries[before].Message)
	}
}

func TestACurrentAttemptStillConnects(t *testing.T) {
	a := connecting()

	stub := &stubClient{}
	model, _ := a.Update(connectedMsg{seq: 1, client: stub, addr: "example.org:21"})
	a = model.(App)

	if !a.connected {
		t.Fatal("a current attempt did not connect")
	}
	if a.connecting {
		t.Error("the application is still marked as connecting")
	}
	if stub.disconnected {
		t.Error("a current attempt was disconnected")
	}
}

// U/D transfer whichever side has marked files, regardless of focus. With
// nothing marked there's no file to infer, so they must not silently no-op.
func TestDirectTransferKeysRequireMarkedFiles(t *testing.T) {
	a := NewApp(nil, false, nil)
	a.focus = focusLocal

	model, _ := a.Update(tea.KeyPressMsg{Code: 'U', Text: "U"})
	a = model.(App)
	if n := len(a.log.entries); n == 0 || a.log.entries[n-1].Level != LogError {
		t.Fatalf("U with nothing marked in LOCAL did not log an error")
	}

	model, _ = a.Update(tea.KeyPressMsg{Code: 'D', Text: "D"})
	a = model.(App)
	if n := len(a.log.entries); n == 0 || a.log.entries[n-1].Level != LogError {
		t.Fatalf("D with nothing marked in REMOTE did not log an error")
	}
}

// Rendering reaches lipgloss and the bubbles list through several derived
// widths and heights, any one of which can go negative before the others do.
// Sweeping the sizes covers the arithmetic without having to find each one.
func TestRenderingSurvivesAnyTerminalSize(t *testing.T) {
	files := []model.FileInfo{
		{Name: "a-rather-long-file-name-to-truncate.txt"},
		{Name: "dir", Type: model.FileTypeDir},
	}

	for w := 0; w <= 24; w++ {
		for h := 0; h <= 24; h++ {
			t.Run(fmt.Sprintf("%dx%d", w, h), func(t *testing.T) {
				a := NewApp(nil, false, nil)
				a.local = a.local.WithFiles(files, "/a/deep/enough/path/to/be/truncated")
				a.remote = a.remote.WithFiles(files, "/another/path")

				model, _ := a.Update(tea.WindowSizeMsg{Width: w, Height: h})
				_ = model.(App).View()
			})
		}
	}
}
