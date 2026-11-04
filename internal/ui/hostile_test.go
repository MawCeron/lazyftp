package ui

import (
	"bytes"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/MawCeron/lazyftp/internal/model"
	"github.com/MawCeron/lazyftp/internal/shared"
)

const hostileName = "evil\x1b]52;c;ZXZpbA==\x07\x1b[2J\u009b0m\rname.txt"

func TestAHostileFileNameIsNotDrawnAsEscapeSequences(t *testing.T) {
	file := model.FileInfo{Name: hostileName, Type: model.FileTypeFile, Size: 5}
	items := []list.Item{fileItem{file: file}}
	delegate := fileDelegate{}
	l := list.New(items, delegate, 80, 10)

	var buf bytes.Buffer
	delegate.Render(&buf, l, 0, items[0])

	requireNoHostileBytes(t, "a file row", buf.String())
}

func TestAHostilePathIsNotDrawnAsEscapeSequences(t *testing.T) {
	files := []model.FileInfo{{Name: "a.txt"}}
	dir := "/srv/" + hostileName

	p, _ := NewPanel("Remote", true).WithFiles(files, dir)
	p = p.SetSize(80, 12)

	requireNoHostileBytes(t, "the path line", p.View(80, 12, true, diffMarks{}))

	filtering, _ := p.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	if filtering.list.FilterState() == list.Unfiltered {
		t.Fatal("/ did not start a filter")
	}
	requireNoHostileBytes(t, "the path line while filtering", filtering.View(80, 12, true, diffMarks{}))
}

func TestAHostileNameIsNotDrawnWhileRenamingOrDeleting(t *testing.T) {
	files := []model.FileInfo{{Name: hostileName, Type: model.FileTypeFile}}
	p, _ := NewPanel("Remote", true).WithFiles(files, "/srv")
	p = p.SetSize(80, 12)

	renaming, _ := p.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	if !renaming.renaming {
		t.Fatal("F2 did not start a rename")
	}
	requireNoHostileBytes(t, "the rename prompt", renaming.View(80, 12, true, diffMarks{}))

	deleting, _ := p.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if !deleting.deleting {
		t.Fatal("d did not start a delete")
	}
	requireNoHostileBytes(t, "the delete prompt", deleting.View(80, 12, true, diffMarks{}))
}

func TestAHostileNameIsNotDrawnInTheFileInfoBox(t *testing.T) {
	out := fileInfoView(model.FileInfo{Name: hostileName, Size: 5}, 100)

	requireNoHostileBytes(t, "the file info box", out)
}

func TestAHostileNameIsNotDrawnInTheProcessesPanel(t *testing.T) {
	row := renderTransfer(shared.Transfer{ID: 1, Filename: hostileName, Total: 10}, 60)

	requireNoHostileBytes(t, "a transfer row", row)
}
