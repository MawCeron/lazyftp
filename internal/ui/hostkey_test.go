package ui

import (
	"strings"
	"testing"
)

func pendingHostKey() *hostKeyPromptMsg {
	return &hostKeyPromptMsg{
		host:        "nas.lan:22",
		keyType:     "ssh-ed25519",
		fingerprint: "SHA256:abcDEF123",
		reply:       make(chan bool, 1),
	}
}

// Every connection starts from the connection dialog, so that is where the
// question arrives. It used to be drawn only with the focus on a panel, a state
// a connection never reaches, and the user waited on a prompt nobody could see.
func TestTheHostKeyQuestionIsDrawnOverTheConnectionDialog(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width, a.height = 100, 30
	a.focus = focusConnectionBar
	a.connecting = true
	a.hostKey = pendingHostKey()

	got := a.render()
	for _, want := range []string{"Unknown Host", "nas.lan:22", "ssh-ed25519", "SHA256:abcDEF123"} {
		if !strings.Contains(got, want) {
			t.Errorf("the screen is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Protocol:") {
		t.Error("the connection dialog is still on screen under the question")
	}
}

func TestTheFooterOffersTheAnswerKeysWhileConnecting(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width = 120
	a.focus = focusConnectionBar
	a.connecting = true
	a.hostKey = pendingHostKey()

	got := a.hintsView()
	if !strings.Contains(got, "trust host") || strings.Contains(got, "abandon") {
		t.Errorf("footer %q should offer y and n, not \"esc abandon\"", got)
	}
}

func TestTheHostKeyQuestionStillShowsOverThePanels(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width, a.height = 100, 30
	a.focus = focusLocal
	a.hostKey = pendingHostKey()

	if got := a.render(); !strings.Contains(got, "Unknown Host") {
		t.Errorf("a question raised while a panel has the focus is not drawn:\n%s", got)
	}
}
