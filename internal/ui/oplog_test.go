package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/MawCeron/lazyftp/internal/model"
)

type opClient struct {
	stubClient
	badPaths map[string]bool
	listErr  error
}

func (o *opClient) Delete(path string, isDir bool) error {
	if o.badPaths[path] {
		return errors.New("permission denied")
	}
	return nil
}
func (o *opClient) List(path string) ([]model.FileInfo, error) { return nil, o.listErr }

func logged(a App) string {
	var b strings.Builder
	for _, e := range a.log.entries {
		b.WriteString(e.Message + "\n")
	}
	return b.String()
}

func TestOfflineActionsSayWhichActionWasBlocked(t *testing.T) {
	a := NewApp(nil, false, nil, "dev", false)
	for want, run := range map[string]func(App) (App, tea.Cmd){
		"create a directory": func(a App) (App, tea.Cmd) { return a.handleMkdir(mkdirMsg{Panel: "Remote", Path: "/x"}) },
		"rename": func(a App) (App, tea.Cmd) {
			return a.handleRename(renameMsg{Panel: "Remote", OldPath: "/a", NewPath: "/b"})
		},
		"delete":   func(a App) (App, tea.Cmd) { return a.handleDelete(deleteMsg{Panel: "Remote"}) },
		"transfer": func(a App) (App, tea.Cmd) { return a.handleTransfer(TransferMsg{}) },
	} {
		got, _ := run(a)
		if l := lastLog(got); l != "No active connection: cannot "+want {
			t.Errorf("%s: log %q", want, l)
		}
	}
}

func TestMkdirConfirmsAndNamesThePathOnFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "new")

	msg, ok := mkdirLocal(target, dir)().(opDoneMsg)
	if !ok || msg.Message != "Local: created directory "+target {
		t.Fatalf("success message: %+v", msg)
	}
	a := NewApp(nil, false, nil, "dev", false)
	got, _ := a.Update(msg)
	if !strings.Contains(logged(got.(App)), "created directory "+target) {
		t.Errorf("confirmation not logged:\n%s", logged(got.(App)))
	}

	fail := mkdirLocal(target, dir)().(LogMsg)
	if !strings.Contains(fail.Message, target) || fail.Level != LogError {
		t.Errorf("failure does not name the path: %+v", fail)
	}
}

func TestRenameConfirmsAndNamesBothPathsOnFailure(t *testing.T) {
	dir := t.TempDir()
	from, to := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	os.WriteFile(from, nil, 0o644)

	if msg, ok := renameLocal(from, to, dir)().(opDoneMsg); !ok || msg.Message != "Local: renamed "+from+" to "+to {
		t.Fatalf("success message: %+v", msg)
	}
	fail := renameLocal(from, to, dir)().(LogMsg) // a.txt is gone now
	if !strings.Contains(fail.Message, from) || !strings.Contains(fail.Message, to) {
		t.Errorf("failure does not name both paths: %q", fail.Message)
	}

	remote := renameRemote(&stubClient{}, "/srv/a", "/srv/b", "/srv")().(opDoneMsg)
	if remote.Message != "Remote: renamed /srv/a to /srv/b" {
		t.Errorf("remote message %q", remote.Message)
	}
}

func TestDeleteLogsHowManyWereDeleted(t *testing.T) {
	c := &opClient{badPaths: map[string]bool{"/srv/locked": true}}
	targets := []deleteTarget{{Path: "/srv/a"}, {Path: "/srv/locked"}, {Path: "/srv/c"}}

	done := deleteRemote(c, targets, "/srv")().(deleteDoneMsg)
	got, _ := NewApp(nil, false, nil, "dev", false).Update(done)
	log := logged(got.(App))
	if !strings.Contains(log, "Remote: 2 of 3 deleted") || !strings.Contains(log, "Error deleting locked: permission denied") {
		t.Errorf("log:\n%s", log)
	}
	if e := got.(App).log.entries; e[len(e)-1].Level != LogError {
		t.Error("a partial delete is reported as a success")
	}

	dir := t.TempDir()
	f := filepath.Join(dir, "x")
	os.WriteFile(f, nil, 0o644)
	got, _ = NewApp(nil, false, nil, "dev", false).Update(deleteLocal([]deleteTarget{{Path: f}}, dir)())
	if l := lastLog(got.(App)); !strings.Contains(logged(got.(App)), "Local: 1 of 1 deleted") {
		t.Errorf("log %q", l)
	}
}

func TestListingErrorsNameTheDirectory(t *testing.T) {
	remote := loadRemoteDir(&opClient{listErr: errors.New("boom")}, "/srv/data")().(LogMsg)
	if !strings.Contains(remote.Message, "/srv/data") || !strings.Contains(remote.Message, "boom") {
		t.Errorf("remote: %q", remote.Message)
	}
	missing := filepath.Join(t.TempDir(), "gone")
	local := loadLocalDir(missing)().(LogMsg)
	if !strings.Contains(local.Message, missing) {
		t.Errorf("local: %q", local.Message)
	}
}
