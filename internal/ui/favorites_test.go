package ui

import (
	"charm.land/lipgloss/v2"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/client"
	"github.com/MawCeron/lazyftp/internal/config"
	"github.com/zalando/go-keyring"
)

var (
	ctrlS = tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
	ctrlO = tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl}
	enter = tea.KeyPressMsg{Code: tea.KeyEnter}
	esc   = tea.KeyPressMsg{Code: tea.KeyEsc}
)

var fav = config.Connection{Name: "nas", Host: "nas.lan", User: "ana", Port: 2222, Protocol: "SFTP"}

func fillField(bar ConnectionBar, f connField, s string) ConnectionBar {
	bar.inputs[f].SetValue(s)
	return bar
}

func TestSaveFlowEmitsFavoriteWithPassword(t *testing.T) {
	bar := NewConnectionBar()
	bar = fillField(bar, fieldHost, "nas.lan")
	bar = fillField(bar, fieldUser, "ana")
	bar = fillField(bar, fieldPass, "hunter2")
	bar.protocol = client.SFTP

	bar, _ = bar.Update(ctrlS)
	if bar.mode != modeSave || bar.name.Value() != "nas.lan" || !bar.remember {
		t.Fatalf("save mode: mode=%v name=%q remember=%v", bar.mode, bar.name.Value(), bar.remember)
	}
	bar, cmd := bar.Update(enter)
	got, ok := cmd().(saveFavoriteMsg)
	if bar.mode != modeForm || !ok || got.Conn.Port != 22 || got.Conn.Protocol != "SFTP" || got.Pass != "hunter2" || !got.Remember {
		t.Fatalf("mode=%v msg=%+v", bar.mode, got)
	}
}

func TestSaveNeedsAHost(t *testing.T) {
	bar, _ := NewConnectionBar().Update(ctrlS)
	if bar.mode != modeForm {
		t.Fatal("opened the save prompt with no host")
	}
}

func TestPickingAFavoriteFillsTheFormAndKeyringPassword(t *testing.T) {
	keyring.MockInit()
	config.SetSecret(fav, "hunter2")

	bar := NewConnectionBar().SetFavorites([]config.Connection{fav})
	bar, _ = bar.Update(ctrlO)
	if bar.mode != modeList {
		t.Fatal("favorites did not open")
	}
	bar, _ = bar.Update(enter)
	if bar.mode != modeForm || bar.protocol != client.SFTP ||
		bar.inputs[fieldHost].Value() != "nas.lan" || bar.inputs[fieldPort].Value() != "2222" ||
		bar.inputs[fieldPass].Value() != "hunter2" {
		t.Fatalf("form not filled: %+v", bar.inputs)
	}
}

func TestEscInsideFavoritesKeepsTheDialogOpen(t *testing.T) {
	a := NewApp(func() *tea.Program { return nil }, false, nil, "dev", false).
		WithConfig("", config.Config{Connections: []config.Connection{fav}}, nil)
	model, _ := a.Update(ctrlO)
	model, _ = model.(App).Update(esc)
	a = model.(App)
	if a.focus != focusConnectionBar || a.connBar.mode != modeForm {
		t.Fatalf("focus=%v mode=%v", a.focus, a.connBar.mode)
	}
}

func TestAppSavesAndDeletesFavorites(t *testing.T) {
	keyring.MockInit()
	a := NewApp(func() *tea.Program { return nil }, false, nil, "dev", false)

	model, _ := a.Update(saveFavoriteMsg{Conn: fav, Pass: "hunter2", Remember: true})
	a = model.(App)
	if len(a.cfg.Connections) != 1 || len(a.connBar.favorites) != 1 {
		t.Fatalf("not saved: %+v", a.cfg.Connections)
	}
	if s, _ := config.Secret(fav); s != "hunter2" {
		t.Fatalf("password not in keyring: %q", s)
	}

	twin := fav
	twin.Name = "nas-copy"
	model, _ = a.Update(saveFavoriteMsg{Conn: twin})
	a = model.(App)
	model, _ = a.Update(deleteFavoriteMsg{Name: "nas"})
	a = model.(App)
	if s, _ := config.Secret(fav); s != "hunter2" {
		t.Fatal("deleting one favorite removed a password another still uses")
	}

	model, _ = a.Update(deleteFavoriteMsg{Name: "nas-copy"})
	a = model.(App)
	if len(a.cfg.Connections) != 0 {
		t.Fatalf("not deleted: %+v", a.cfg.Connections)
	}
	if _, err := config.Secret(fav); err == nil {
		t.Fatal("password outlived its last favorite")
	}
}

func TestPasswordIsNotStoredUnlessRemembered(t *testing.T) {
	keyring.MockInit()
	a := NewApp(func() *tea.Program { return nil }, false, nil, "dev", false)
	a.Update(saveFavoriteMsg{Conn: fav, Pass: "hunter2", Remember: false})
	if _, err := config.Secret(fav); err == nil {
		t.Fatal("password stored without being asked")
	}
}

func TestDialogsFitTheirBox(t *testing.T) {
	bar := fillField(NewConnectionBar(), fieldHost, "nas.lan").SetFavorites([]config.Connection{fav})
	saving, _ := bar.Update(ctrlS)
	listing, _ := bar.Update(ctrlO)
	replacing := bar.SetFavorites([]config.Connection{{Name: "nas.lan", Host: "nas.lan", User: "ana", Port: 22, Protocol: "FTP"}})
	replacing, _ = replacing.Update(ctrlS)
	replacing.name.SetValue("nas.lan")
	replacing, _ = replacing.Update(enter)
	for name, v := range map[string]string{"form": bar.View(80), "save": saving.View(80), "list": listing.View(80), "replace": replacing.View(80)} {
		for i, line := range strings.Split(v, "\n") {
			if w := lipgloss.Width(line); w != 56 {
				t.Errorf("%s line %d is %d wide, want 56: %q", name, i, w, line)
			}
		}
	}
	if !strings.Contains(bar.View(80), "Esc cancel") {
		t.Error("form hint was cut")
	}
}

func TestSavingOverAnExistingNameAsksFirst(t *testing.T) {
	bar := NewConnectionBar().SetFavorites([]config.Connection{fav})
	bar = fillField(bar, fieldHost, "nas.lan")
	bar, _ = bar.Update(ctrlS)
	if got := bar.name.Value(); got != "nas.lan" {
		t.Fatalf("suggested %q", got)
	}

	bar.name.SetValue("nas")
	bar, cmd := bar.Update(enter)
	if bar.mode != modeReplace || cmd != nil {
		t.Fatalf("mode=%v, sent=%v: overwrote without asking", bar.mode, cmd != nil)
	}
	back, _ := bar.Update(esc)
	if back.mode != modeSave {
		t.Fatal("n/esc did not return to the name")
	}
	bar, cmd = bar.Update(keyMsg("y"))
	if bar.mode != modeForm || cmd == nil {
		t.Fatal("y did not replace")
	}
}

func TestSuggestedNameNeverCollides(t *testing.T) {
	bar := NewConnectionBar().SetFavorites([]config.Connection{{Name: "nas.lan"}, {Name: "nas.lan (2)"}})
	bar = fillField(bar, fieldHost, "nas.lan")
	bar, _ = bar.Update(ctrlS)
	if got := bar.name.Value(); got != "nas.lan (3)" {
		t.Fatalf("suggested %q", got)
	}
}

func TestReplacingChangesServerReleasesItsPassword(t *testing.T) {
	keyring.MockInit()
	a := NewApp(func() *tea.Program { return nil }, false, nil, "dev", false)
	model, _ := a.Update(saveFavoriteMsg{Conn: fav, Pass: "hunter2", Remember: true})
	moved := fav
	moved.Host = "other.lan"
	model, _ = model.(App).Update(saveFavoriteMsg{Conn: moved})
	if _, err := config.Secret(fav); err == nil {
		t.Fatal("the old server's password outlived the favorite that owned it")
	}
	if n := len(model.(App).cfg.Connections); n != 1 {
		t.Fatalf("%d favorites, want 1", n)
	}
}

func TestFooterFollowsTheDialogMode(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.width = 100
	a.focus = focusConnectionBar
	for mode, want := range map[barMode]string{modeList: "fill", modeHistory: "fill", modeSave: "password", modeReplace: "replace"} {
		a.connBar.mode = mode
		got := a.hintsView()
		if !strings.Contains(got, want) || strings.Contains(got, "close") {
			t.Errorf("mode %d: footer %q, want %q and no \"close\"", mode, got, want)
		}
	}
}

func TestConnectingRecordsAndRecentFillsTheForm(t *testing.T) {
	keyring.MockInit()
	a := NewApp(nil, false, nil, "dev", false)
	a.connecting, a.connectSeq = true, 1
	model, _ := a.Update(connectedMsg{seq: 1, client: &stubClient{}, host: "nas.lan", port: 2222, addr: "nas.lan:2222", user: "ana", protocol: client.SFTP})
	a = model.(App)
	if len(a.history) != 1 || len(a.connBar.recent) != 1 {
		t.Fatalf("not recorded: %+v", a.history)
	}

	a.focus = focusConnectionBar
	model, _ = a.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	model, _ = model.(App).Update(enter)
	bar := model.(App).connBar
	if bar.mode != modeForm || bar.inputs[fieldHost].Value() != "nas.lan" || bar.inputs[fieldPort].Value() != "2222" || bar.protocol != client.SFTP {
		t.Fatalf("form not filled: %+v", bar.inputs)
	}
}

func TestAnAbandonedConnectionIsNotRecorded(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	a.connectSeq = 2
	model, _ := a.Update(connectedMsg{seq: 1, client: &stubClient{}, host: "x"})
	if len(model.(App).history) != 0 {
		t.Fatal("recorded an attempt nobody is waiting for")
	}
}
